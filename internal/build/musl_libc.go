package build

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type MuslLibc struct {
	SourceBuilder
	FilesystemOutputBuilder
	ToolchainRequiredBuilder
	GitRefBuilder
	RootFSBuilder
	Triple *utils.Triplet

	// Vars for validation checking
	sourceVersion string
}

func (ml *MuslLibc) CheckHostRequirements() error {
	err := ml.CheckToolsExist()
	if err != nil {
		return trace.Wrap(err, "failed to verify that all required toolchain tools exist")
	}

	return nil
}

func (ml *MuslLibc) Build(ctx context.Context) error {
	slog.Info("Starting Musl libc build")
	repo := git_source.NewMuslGitRepo(ml.SourceDirectoryPath, ml.GitRef)
	sourcePath := repo.FullDownloadPath()

	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, ml.OutputDirectoryPath)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup for Musl build")
	}
	ml.OutputDirectoryPath = outputDirectory.Path

	err = ml.runMuslConfigure(sourcePath, buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to configure musl libc")
	}

	err = ml.runMuslMake(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to build musl libc")
	}

	// Record values for build verification
	fileContents, err := os.ReadFile(path.Join(sourcePath, "VERSION"))
	if err != nil {
		return trace.Wrap(err, "failed to read Musl VERSION file")
	}
	ml.sourceVersion = strings.TrimSpace(string(fileContents))

	return nil
}

func (ml *MuslLibc) runMuslMake(buildDirectoryPath string) error {
	_, err := runners.Run(&runners.Make{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectoryPath,
			Options: []*runners.GenericRunnerOptions{
				ml.ToolchainRequiredBuilder.GetGenericRunnerOptions(),
			},
		},
		Path:    ".",
		Targets: []string{"install"},
		Variables: map[string]args.IValue{
			"DESTDIR": args.StringValue(path.Join(ml.OutputDirectoryPath, "usr")), // This must be set so that all files are installed/written to the output directory
		},
	})

	if err != nil {
		return trace.Wrap(err, "musl libc make build failed")
	}

	return nil
}

func (ml *MuslLibc) runMuslConfigure(sourceDirectoryPath, buildDirectoryPath string) error {
	_, err := runners.Run(&runners.Configure{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectoryPath,
			Options: []*runners.GenericRunnerOptions{
				ml.ToolchainRequiredBuilder.GetGenericRunnerOptions(),
			},
		},
		Options: []*runners.ConfigureOptions{
			// Install subdirectory is empty because Musl builds need it set when calling the makefile
			// so that all files install under the correct directory. The makefile does not strictly
			// follow GNU configure requirements.
			// This path is relative to DESTDIR, which is set when calling make
			ml.ToolchainRequiredBuilder.GetConfigurenOptions(),
			ml.RootFSBuilder.GetConfigurenOptions(),
			{
				AdditionalArgs: map[string]args.IValue{
					"--prefix": args.StringValue("/"), //  Path is relative to "DESTDIR" which is set when invoking Make
					"--srcdir": args.StringValue(sourceDirectoryPath),
				},
			},
		},
		ConfigurePath: path.Join(sourceDirectoryPath, "configure"),
		HostTriplet:   ml.Triple,
		TargetTriplet: ml.Triple,
	})

	if err != nil {
		return trace.Wrap(err, "failed to configure musl")
	}

	return nil
}

func (ml *MuslLibc) VerifyBuild(ctx context.Context) error {
	libcPath := path.Join(ml.OutputDirectoryPath, "usr", "lib", "libc.so")
	isValid, version, err := (&runners.VersionChecker{
		CommandRunner: runners.CommandRunner{
			Command: libcPath,
			Arguments: []string{
				"--version",
			},
		},
		IgnoreErrorExit: true,
		UseStdErr:       true,
		VersionRegex:    fmt.Sprintf("(?m)^Version %s$", runners.SemverRegex),
		VersionChecker:  runners.ExactSemverChecker(ml.sourceVersion),
	}).IsValidVersion()
	if err != nil {
		return trace.Wrap(err, "failed to retreive built musl libc version")
	}

	if !isValid {
		return trace.Errorf("built musl libc version %q does not match build version %q", version, ml.sourceVersion)
	}

	err = ml.VerifyTargetElfFile(libcPath)
	if err != nil {
		return trace.Wrap(err, "libc file %q did not match the expected ELF values", libcPath)
	}

	return nil
}
