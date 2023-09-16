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
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type MuslLibc struct {
	SourceBuilder
	FilesystemOutputBuilder
	ToolchainRequiredBuilder
	GitRefBuilder
	Triple                    *utils.Triplet
	KernelHeaderDirectoryPath string

	// Vars for validation checking
	sourceVersion string
}

func (ml *MuslLibc) CheckHostRequirements() error {
	requiredToolchainCommands := []string{
		"clang",
		"clang++",
	}

	for i := range requiredToolchainCommands {
		requiredToolchainCommands[i] = ml.GetPathForTool(requiredToolchainCommands[i])
	}

	err := runners.CheckRequiredCommandsExist(requiredToolchainCommands)
	if err != nil {
		return trace.Wrap(err, "failed to verify that all required commands exist")
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
			EnvironmentVariables: map[string]string{
				"PATH": fmt.Sprintf("%s%c%s", ml.ToolchainPath, os.PathListSeparator, os.Getenv("PATH")), // Path is set to ensure that builds use toolchain tools when not prefixed properly
			},
		},
		Path:    ".",
		Targets: []string{"install"},
		Variables: map[string]string{
			"DESTDIR": ml.OutputDirectoryPath, // This must be set so that all files are installed/written to the output directory
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
		},
		CCompiler:           ml.GetPathForTool("clang"),
		CppCompiler:         ml.GetPathForTool("clang++"),
		InstallPath:         "/", // This path is relative to DESTDIR, which is set when calling make
		SourceDirectoryPath: sourceDirectoryPath,
		ConfigurePath:       path.Join(sourceDirectoryPath, "configure"),
		HostTriplet:         ml.Triple,
		TargetTriplet:       ml.Triple,
		CFlags:              fmt.Sprintf("-I%s", ml.KernelHeaderDirectoryPath),
	})

	if err != nil {
		return trace.Wrap(err, "failed to configure musl")
	}

	return nil
}

func (ml *MuslLibc) VerifyBuild(ctx context.Context) error {
	isValid, version, err := (&runners.VersionChecker{
		CommandRunner: runners.CommandRunner{
			Command: path.Join(ml.OutputDirectoryPath, "lib", "libc.so"),
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

	return nil
}
