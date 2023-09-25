package build

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
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
	DoConfiguration(sourceDirectoryPath, buildDirectoryPath string) error
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
	sourcePath, buildDirectory, err := sb.Setup(ctx)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup builder")
	}

	err = sb.DoConfiguration(sourcePath, buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to configure %s", sb.Name)
	}

	err = sb.DoBuild(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to build %s", sb.Name)
	}

	return nil
}

func (sb *StandardBuilder) Setup(ctx context.Context) (string, *utils.Directory, error) {
	slog.Info(fmt.Sprintf("Starting %s build", sb.Name))
	repo := sb.GetGitRepo(sb.SourceDirectoryPath, sb.GitRef)
	sourcePath := repo.FullDownloadPath()

	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, sb.OutputDirectoryPath)
	if err != nil {
		return "", nil, trace.Wrap(err, "failed to setup for %s build", sb.Name)
	}
	sb.OutputDirectoryPath = outputDirectory.Path

	return sourcePath, buildDirectory, nil
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

func (sb *StandardBuilder) CMakeConfigure(sourceDirectoryPath, buildDirectoryPath, cmakeSubpath string, options ...*runners.CMakeOptions) error {
	_, err := runners.Run(&runners.CMake{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Generator:     "Ninja",
		Path:          path.Join(sourceDirectoryPath, cmakeSubpath),
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

func (sb *StandardBuilder) CMakeConfigureFixPkgconfigPrefix(sourceDirectoryPath, buildDirectoryPath, pkgconfigSubpath, cmakeSubpath string, options ...*runners.CMakeOptions) error {
	err := sb.CMakeConfigure(sourceDirectoryPath, buildDirectoryPath, cmakeSubpath, options...)
	if err != nil {
		return trace.Wrap(err, "failed to run CMake for %s", sb.Name)
	}

	err = updatePkgconfigPrefix(path.Join(buildDirectoryPath, pkgconfigSubpath))
	if err != nil {
		return trace.Wrap(err, "failed to update pkgconfig prefix for %q", sb.Name)
	}

	return nil
}

func (sb *StandardBuilder) GNUConfigure(sourceDirectoryPath, buildDirectoryPath string, flags ...string) error {
	_, err := runners.Run(&runners.Configure{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Options: []*runners.ConfigureOptions{
			sb.ToolchainRequiredBuilder.GetConfigurenOptions(),
			sb.RootFSBuilder.GetConfigurenOptions(),
			{
				AdditionalArgs: map[string]args.IValue{
					"--prefix": args.StringValue("/"), // Path is relative to DESTDIR, set when invoking make
					"--srcdir": args.StringValue(sourceDirectoryPath),
				},
				AdditionalFlags: pie.Map(flags, func(flag string) args.IValue { return args.StringValue(flag) }),
			},
		},
		ConfigurePath: path.Join(sourceDirectoryPath, "configure"),
		HostTriplet:   sb.ToolchainRequiredBuilder.Triplet,
		TargetTriplet: sb.ToolchainRequiredBuilder.Triplet,
	})

	if err != nil {
		return trace.Wrap(err, "failed to run configure script for %s", sb.Name)
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

func (sb *StandardBuilder) MakeBuild(makefileDirectoryPath, outputDirectoryPath string, makeVars map[string]args.IValue, targets ...string) error {
	for _, target := range targets {
		_, err := runners.Run(&runners.Make{
			GenericRunner: sb.getGenericRunner(makefileDirectoryPath),
			Path:          ".",
			Targets:       []string{target},
			Variables:     makeVars,
		})

		if err != nil {
			return trace.Wrap(err, "%s make build failed for target %q", sb.Name, target)
		}
	}

	return nil
}
