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

func (sb *StandardBuilder) CMakeConfigure(buildDirectoryPath string, options ...*runners.CMakeOptions) error {
	return trace.Wrap(sb.CMakeConfigureWithPath(buildDirectoryPath, sb.SourceDirectoryPath, options...))
}

func (sb *StandardBuilder) CMakeConfigureWithPath(buildDirectoryPath, cmakePath string, options ...*runners.CMakeOptions) error {
	_, err := runners.Run(&runners.CMake{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Generator:     "Ninja",
		Path:          cmakePath,
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

func (sb *StandardBuilder) GNUConfigure(buildDirectoryPath string, flags ...string) error {
	err := sb.GNUConfigureWithSrc(buildDirectoryPath, sb.SourceDirectoryPath, flags...)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

func (sb *StandardBuilder) AutogenConfigure(buildDirectoryPath string, flags ...string) error {
	err := sb.Autogen(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run autogen for %s build", sb.Name)
	}

	err = sb.GNUConfigureWithSrc(buildDirectoryPath, buildDirectoryPath, flags...)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// Run ./bootstrap && ./autogen && ./configure <flags>. This is usually used by GNU tools.
func (sb *StandardBuilder) BootstrapAutogenConfigure(buildDirectoryPath string, flags ...string) error {
	err := sb.Bootstrap(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to bootstrap build directory %q", buildDirectoryPath)
	}

	err = sb.autogenNoCopy(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run autogen on copied sources")
	}

	err = sb.GNUConfigureWithSrc(buildDirectoryPath, buildDirectoryPath, flags...)
	if err != nil {
		return trace.Wrap(err, "failed to run configure on copied sources")
	}

	return nil
}

func (sb *StandardBuilder) GNUConfigureWithSrc(buildDirectoryPath, sourceDirectoryPath string, flags ...string) error {
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

// Searches for /<build directory path>/**/pkgconfig/**/*.pc files and updates the included paths to be relative
// to the root directory.
// For example, `prefix=/tmp/output/package/usr` will be rewritten as `prefix=/usr`.
func (sb *StandardBuilder) UpdatePkgconfigsPrefixes(searchDirectoryPath string) error {
	err := filepath.WalkDir(searchDirectoryPath, func(fsPath string, fsEntry fs.DirEntry, err error) error {
		if err != nil {
			return trace.Wrap(err, "failed to walk dir %q", fsPath)
		}

		// Skip directories
		if fsEntry.IsDir() {
			return nil
		}

		buildDirectoryRelativePath, err := filepath.Rel(searchDirectoryPath, fsPath)
		if err != nil {
			// This should normally only be hit if there is a bug
			return trace.Wrap(err, "failed to get path %q relative to the build directory path %q", fsPath, searchDirectoryPath)
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
		return trace.Wrap(err, "failed to walk over and fix all pkgconfig files in %q", searchDirectoryPath)
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
			"bindir":      "${prefix}/bin",
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

func (sb *StandardBuilder) MesonSetup(buildDirectoryPath string, options ...*runners.MesonOptions) error {
	_, err := runners.Run(&runners.Meson{
		GenericRunner:       sb.getGenericRunner(buildDirectoryPath), // This does not nescessarily need to be set to the build directory,
		Backend:             "Ninja",
		SourceDirectoryPath: sb.SourceDirectoryPath,
		BuildDirectoryPath:  buildDirectoryPath,
		Options: append(
			[]*runners.MesonOptions{
				sb.FilesystemOutputBuilder.GetMesonOptions(),
				sb.ToolchainRequiredBuilder.GetMesonOptions(),
				sb.RootFSBuilder.GetMesonOptions(),
			},
			options...,
		),
	})

	if err != nil {
		return trace.Wrap(err, "failed to create perform setup with meson for %s", sb.Name)
	}

	return nil
}

func (sb *StandardBuilder) NinjaBuild(buildDirectoryPath string, buildTargets ...string) error {
	return sb.genericNinjaBuild(sb.getGenericRunner(buildDirectoryPath), buildDirectoryPath, buildTargets...)
}

// This is a version of a Ninja build with extra meson-specific variables set
func (sb *StandardBuilder) MesonNinjaBuild(buildDirectoryPath string, buildTargets ...string) error {
	baseRunner := sb.getGenericRunner(buildDirectoryPath) // This does not nescessarily need to be set to the build directory
	baseRunner.Options = append(baseRunner.Options, &runners.GenericRunnerOptions{
		EnvironmentVariables: map[string]args.IValue{
			"DESTDIR": args.StringValue(sb.OutputDirectoryPath), // This has to be set as an environment variable specifically
		},
	})

	return sb.genericNinjaBuild(baseRunner, buildDirectoryPath, buildTargets...)
}

func (sb *StandardBuilder) genericNinjaBuild(baseRunner runners.GenericRunner, buildDirectoryPath string, buildTargets ...string) error {
	if len(buildTargets) == 0 {
		buildTargets = append(buildTargets, "install")
	}

	_, err := runners.Run(runners.CommandRunner{
		Command:       "ninja",
		Arguments:     buildTargets,
		GenericRunner: baseRunner,
	})
	if err != nil {
		return trace.Wrap(err, "failed to build %s", sb.Name)
	}

	err = sb.UpdatePkgconfigsPrefixes(sb.OutputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to update pkgconfig prefixes")
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

func (sb *StandardBuilder) Bootstrap(buildDirectoryPath string) error {
	err := sb.CopyToBuildDirectory(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to copy source directory %q to build directory %q", sb.SourceDirectoryPath, buildDirectoryPath)
	}

	err = sb.bootstrapNoCopy(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run bootstrap on copied sources")
	}

	return nil
}

func (sb *StandardBuilder) bootstrapNoCopy(buildDirectoryPath string) error {
	var bootstrapScriptPath string
	for _, possibleBootstrapFile := range []string{"bootstrap", "bootstrap.sh"} {
		possibleBootstrapFilePath := path.Join(buildDirectoryPath, possibleBootstrapFile)
		doesExists, err := utils.DoesFilesystemPathExist(possibleBootstrapFilePath)
		if err != nil {
			return trace.Wrap(err, "failed to check if bootstrap file %q exists", possibleBootstrapFilePath)
		}

		if doesExists {
			bootstrapScriptPath = possibleBootstrapFilePath
			break
		}
	}

	if bootstrapScriptPath == "" {
		return trace.Errorf("failed to find bootstrap file in %q", buildDirectoryPath)
	}

	_, err := runners.Run(&runners.CommandRunner{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Command:       bootstrapScriptPath,
	})
	if err != nil {
		return trace.Wrap(err, "command bootstrap failed in build directory %q", buildDirectoryPath)
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

	err = sb.autogenNoCopy(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run autogen on copied sources")
	}

	return nil
}

func (sb *StandardBuilder) autogenNoCopy(buildDirectoryPath string) error {
	_, err := runners.Run(&runners.CommandRunner{
		GenericRunner: sb.getGenericRunner(buildDirectoryPath),
		Command:       path.Join(buildDirectoryPath, "autogen.sh"),
	})
	if err != nil {
		return trace.Wrap(err, "command autogen.sh failed in build directory %q", buildDirectoryPath)
	}

	return nil
}

func (sb *StandardBuilder) RunLibtool(buildDirectoryPath string) error {
	_, err := runners.Run(runners.CommandRunner{
		Command: path.Join(buildDirectoryPath, "libtool"),
		Arguments: []string{
			"--finish",
			path.Join(sb.OutputDirectoryPath, "usr", "lib"),
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to run libtool --finish on output lib directory")
	}

	return nil
}
