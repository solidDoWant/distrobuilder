package build

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type IStandardBuilder interface {
	GetGitRepo(repoDirectoryPath, ref string) *source.GitRepo
	DoConfiguration(buildDirectoryPath string) error
	DoBuild(buildDirectoryPath string) error
}

// Most builders outside of the initial cross-compile toolchain
// and libc builders will follow the same general pattern.
// This builder abstracts it to reduce the boilerplate and
// maintenance required to maintain each builder.
type StandardBuilder struct {
	IStandardBuilder
	SourceBuilder
	FilesystemOutputBuilder
	ToolchainRequiredBuilder
	GitRefBuilder
	RootFSBuilder

	// Variables for building
	Name string

	// Variables for build verification
	BinariesToCheck []string
}

func (sb *StandardBuilder) CheckHostRequirements() error {
	err := sb.CheckToolsExist()
	if err != nil {
		return trace.Wrap(err, "failed to verify that all required toolchain tools exist")
	}

	return nil
}

func (sb *StandardBuilder) Build(ctx context.Context) error {
	buildDirectory, err := sb.Setup(ctx)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup builder")
	}

	err = sb.DoConfiguration(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to configure %s", sb.Name)
	}

	err = sb.DoBuild(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to build %s", sb.Name)
	}

	return nil
}

func (sb *StandardBuilder) Setup(ctx context.Context) (*utils.Directory, error) {
	slog.Info(fmt.Sprintf("Starting %s build", sb.Name))
	repo := sb.GetGitRepo(sb.SourceDirectoryPath, sb.GitRef)
	sb.SourceDirectoryPath = repo.FullDownloadPath()

	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, sb.OutputDirectoryPath)
	if err != nil {
		return nil, trace.Wrap(err, "failed to setup for %s build", sb.Name)
	}
	sb.OutputDirectoryPath = outputDirectory.Path

	return buildDirectory, nil
}

func (sb *StandardBuilder) VerifyBuild(ctx context.Context) error {
	for _, binaryPath := range sb.BinariesToCheck {
		err := sb.VerifyTargetElfFile(path.Join(sb.OutputDirectoryPath, binaryPath))
		if err != nil {
			return trace.Wrap(err, "built file %q did not match the expected ELF values", binaryPath)
		}
	}

	return nil
}

func (sb *StandardBuilder) getGenericRunner(workingDirectory string) runners.GenericRunner {
	return runners.GenericRunner{
		WorkingDirectory: workingDirectory,
		Options: []*runners.GenericRunnerOptions{
			sb.ToolchainRequiredBuilder.GetGenericRunnerOptions(),
			sb.RootFSBuilder.GetGenericRunnerOptions(),
		},
	}
}

func (sb *StandardBuilder) CMakeConfigureOnly(buildDirectoryPath, cmakeSubpath string, options ...*runners.CMakeOptions) error {
	_, err := runners.Run(&runners.CMake{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Generator:     "Ninja",
		Path:          path.Join(sb.SourceDirectoryPath, cmakeSubpath),
		Options: append(
			[]*runners.CMakeOptions{
				sb.FilesystemOutputBuilder.GetCMakeOptions("usr"),
				sb.ToolchainRequiredBuilder.GetCMakeOptions(),
				sb.RootFSBuilder.GetCMakeOptions(),
			},
			options...,
		),
	})

	if err != nil {
		return trace.Wrap(err, "failed to create generator file for %s", sb.Name)
	}

	return nil
}

func (sb *StandardBuilder) CMakeConfigure(buildDirectoryPath, cmakeSubpath string, options ...*runners.CMakeOptions) error {
	err := sb.CMakeConfigureOnly(buildDirectoryPath, cmakeSubpath, options...)
	if err != nil {
		return trace.Wrap(err, "failed to run CMake for %s", sb.Name)
	}

	err = updatePkgconfigsPrefixes(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to update pkgconfig prefix for %q", sb.Name)
	}

	return nil
}

func (sb *StandardBuilder) GNUConfigure(buildDirectoryPath string, flags ...string) error {
	_, err := runners.Run(&runners.Configure{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Options: []*runners.ConfigureOptions{
			sb.ToolchainRequiredBuilder.GetConfigurenOptions(),
			sb.RootFSBuilder.GetConfigurenOptions(),
			{
				AdditionalArgs: map[string]args.IValue{
					"--prefix": args.StringValue("/"), // Path is relative to DESTDIR, set when invoking make
					"--srcdir": args.StringValue(sb.SourceDirectoryPath),
				},
				AdditionalFlags: pie.Map(flags, func(flag string) args.IValue { return args.StringValue(flag) }),
			},
		},
		ConfigurePath: path.Join(sb.SourceDirectoryPath, "configure"),
		HostTriplet:   sb.ToolchainRequiredBuilder.Triplet,
		TargetTriplet: sb.ToolchainRequiredBuilder.Triplet,
	})

	if err != nil {
		return trace.Wrap(err, "failed to run configure script for %s", sb.Name)
	}

	return nil
}

func updatePkgconfigsPrefixes(buildDirectoryPath string) error {
	err := filepath.WalkDir(buildDirectoryPath, func(fsPath string, fsEntry fs.DirEntry, err error) error {
		if err != nil {
			return trace.Wrap(err, "failed to walk dir %q", fsPath)
		}

		// Skip directories
		if fsEntry.IsDir() {
			return nil
		}

		buildDirectoryRelativePath, err := filepath.Rel(buildDirectoryPath, fsPath)
		if err != nil {
			// This should normally only be hit if there is a bug
			return trace.Wrap(err, "failed to get path %q relative to the build directory path %q", fsPath, buildDirectoryPath)
		}

		if !slices.Contains(strings.Split(buildDirectoryRelativePath, string(filepath.Separator)), "pkgconfig") {
			return nil
		}

		if filepath.Ext(fsPath) != ".pc" {
			return nil
		}

		slog.Info("Updating pkgconfig prefixes", "path", buildDirectoryRelativePath)

		// At this point the file must be a pkgconfig file
		err = updatePkgconfigPrefix(fsPath)
		if err != nil {
			return trace.Wrap(err, "failed to update package config prefixes for %q", fsPath)
		}

		return nil
	})

	if err != nil {
		return trace.Wrap(err, "failed to walk over and fix all pkgconfig files in %q", buildDirectoryPath)
	}

	return nil
}

func updatePkgconfigPrefix(pkgconfigFilePath string) error {
	// Patch the pkg-config file with the correct prefix.
	// The CMake configuration sets this to CMAKE_INSTALL_PREFIX,
	// which is also needed to set the install path.
	fileContents, err := os.ReadFile(pkgconfigFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to read pkg-config file at %q", pkgconfigFilePath)
	}

	fileLines := strings.Split(string(fileContents), "\n")
	for i, line := range fileLines {
		for key, value := range map[string]string{
			"prefix":      "/usr",
			"exec_prefix": "${prefix}",
			"libdir":      "${prefix}/lib",
			"includedir":  "${prefix}/include",
		} {
			prefix := fmt.Sprintf("%s=", key)
			if strings.HasPrefix(line, prefix) {
				line = fmt.Sprintf("%s%s", prefix, value)
			}
		}

		fileLines[i] = line
	}

	fileContents = []byte(strings.Join(fileLines, "\n"))
	err = os.WriteFile(pkgconfigFilePath, fileContents, 0644)
	if err != nil {
		return trace.Wrap(err, "failed to write pkg-config file at %q", pkgconfigFilePath)
	}

	return nil
}

func (sb *StandardBuilder) NinjaBuild(buildDirectoryPath string, buildTargets ...string) error {
	if len(buildTargets) == 0 {
		buildTargets = append(buildTargets, "install")
	}

	_, err := runners.Run(runners.CommandRunner{
		Command:       "ninja",
		Arguments:     buildTargets,
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
	})
	if err != nil {
		return trace.Wrap(err, "failed to build %s", sb.Name)
	}

	return nil
}

// Produces a built via Make using the provided confiruation. Targets are run in series, not in parallel.
func (sb *StandardBuilder) MakeBuild(makefileDirectoryPath string, makeOptions []*runners.MakeOptions, targets ...string) error {
	for _, target := range targets {
		_, err := runners.Run(&runners.Make{
			GenericRunner: sb.getGenericRunner(makefileDirectoryPath),
			Path:          ".",
			Targets:       []string{target},
			Options:       makeOptions,
		})

		if err != nil {
			return trace.Wrap(err, "%s make build failed for target %q", sb.Name, target)
		}
	}

	return nil
}

func (sb *StandardBuilder) Autogen(buildDirectoryPath string) error {
	// Autogen is somewhat strange and does not support generating files in
	// a separate build directory. To prevent contaminating the source folder,
	// the source contents are first copied to the build directory.
	// Somehow Go does not have a builtin library for copying directories, so
	// use this third party one that should cover most corner cases.
	err := sb.CopyToBuildDirectory(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to copy source directory %q to build directory %q", sb.SourceDirectoryPath, buildDirectoryPath)
	}

	_, err = runners.Run(&runners.CommandRunner{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Command:       path.Join(buildDirectoryPath, "autogen.sh"),
	})
	if err != nil {
		return trace.Wrap(err, "command autogen.sh failed in build directory %q", buildDirectoryPath)
	}

	return nil
}
