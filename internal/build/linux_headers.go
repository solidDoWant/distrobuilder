package build

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type LinuxHeaders struct {
	SourceBuilder
	FilesystemOutputBuilder
	GitRef string

	// Vars for validation checking
	sourceVersion string
}

func (lh *LinuxHeaders) CheckHostRequirements() error {
	requiredToolchainCommands := []string{
		"make",
		"clang", // For version checking of output build
	}

	err := runners.CheckRequiredCommandsExist(requiredToolchainCommands)
	if err != nil {
		return trace.Wrap(err, "failed to verify that all required commands exist")
	}

	return nil
}

func (lh *LinuxHeaders) Build(ctx context.Context) error {
	slog.Info("Starting Linux build (headers only)")
	repo := git_source.NewLinuxGitRepo(lh.SourceDirectoryPath, lh.GitRef)
	sourceDirectory := repo.FullDownloadPath()

	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, lh.OutputDirectoryPath)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup for linux header build")
	}

	lh.OutputDirectoryPath = outputDirectory.Path

	_, err = lh.runLinuxMake(sourceDirectory, buildDirectory.Path, "mrproper")
	if err != nil {
		return trace.Wrap(err, "failed to run make mrproper")
	}

	_, err = lh.runLinuxMake(sourceDirectory, buildDirectory.Path, "headers_install")
	if err != nil {
		return trace.Wrap(err, "failed to run make headers_install")
	}

	// Record data for verification test
	kernelVersion, err := lh.runLinuxMake(sourceDirectory, buildDirectory.Path, "kernelversion")
	if err != nil {
		return trace.Wrap(err, "failed to run make kernelversion")
	}
	lh.sourceVersion = strings.TrimSpace(kernelVersion)

	return nil
}

func (lh *LinuxHeaders) runLinuxMake(sourceDirectoryPath, buildDirectoryPath, buildTarget string) (string, error) {
	result, err := runners.Run(&runners.Make{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectoryPath,
		},
		Path:    sourceDirectoryPath,
		Targets: []string{buildTarget},
		Variables: map[string]string{
			"INSTALL_HDR_PATH": lh.OutputDirectoryPath,
		},
	})

	if err != nil {
		return "", trace.Wrap(err, "linux headers make %s failed", buildTarget)
	}

	return result.Stdout, nil
}

func (lh *LinuxHeaders) VerifyBuild(ctx context.Context) error {
	slog.Info(fmt.Sprintf("Verifying that headers match source version %q", lh.sourceVersion))
	testMacroFile := strings.Join([]string{
		"#define VERSION(major,minor,patch) VERSION_(major,minor,patch)",
		"#define VERSION_(major,minor,patch) major ## . ## minor ## . ## patch",
		"VERSION(LINUX_VERSION_MAJOR, LINUX_VERSION_PATCHLEVEL, LINUX_VERSION_SUBLEVEL)",
	}, "\n")

	result, err := runners.Run(runners.CommandRunner{
		Command: "clang", // Any version of clang works here, not just the cross compiler built by this tool
		Arguments: []string{
			"-E",       // Invoke the preprocessor only
			"-P",       // Print the preprocessor output only
			"-include", // Include the version.h header file fron the built headers
			path.Join(lh.OutputDirectoryPath, "include", "linux", "version.h"),
			"-", // Read input from stdin
		},
		Stdin: testMacroFile,
	})
	if err != nil {
		return trace.Wrap(err, "failed to get built linux header version")
	}

	builtVersion := strings.TrimSpace(result.Stdout)
	if builtVersion != lh.sourceVersion {
		return trace.Errorf("linux header version mismatch: expected %s, got %s", lh.sourceVersion, builtVersion)
	}

	return nil
}
