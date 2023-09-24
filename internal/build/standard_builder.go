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
	"github.com/solidDoWant/distrobuilder/internal/source"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

// Most builders outside of the initial cross-compile toolchain
// and libc builders will follow the same general pattern.
// This builder abstracts it to reduce the boilerplate and
// maintenance required to maintain each builder.
type StandardBuilder struct {
	SourceBuilder
	FilesystemOutputBuilder
	ToolchainRequiredBuilder
	GitRefBuilder
	RootFSBuilder

	// Variables for building
	Name            string
	GitRepo         func(repoDirectoryPath, ref string) *source.GitRepo
	DoConfiguration func(sb *StandardBuilder, sourceDirectoryPath, buildDirectoryPath string) error
	DoBuild         func(sb *StandardBuilder, buildDirectoryPath string) error

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
	slog.Info(fmt.Sprintf("Starting %s build", sb.Name))
	repo := sb.GitRepo(sb.SourceDirectoryPath, sb.GitRef)
	sourcePath := repo.FullDownloadPath()

	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, sb.OutputDirectoryPath)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup for %s build", sb.Name)
	}
	sb.OutputDirectoryPath = outputDirectory.Path

	err = sb.DoConfiguration(sb, sourcePath, buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to configure %s", sb.Name)
	}

	err = sb.DoBuild(sb, buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to build %s", sb.Name)
	}

	return nil
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

func CMakeConfigure(cmakeSubpath string, options ...*runners.CMakeOptions) func(*StandardBuilder, string, string) error {
	return func(sb *StandardBuilder, sourceDirectoryPath, buildDirectoryPath string) error {
		_, err := runners.Run(&runners.CMake{
			GenericRunner: runners.GenericRunner{
				WorkingDirectory:     buildDirectoryPath,
				EnvironmentVariables: sb.GetEnvironmentVariables(),
			},
			Generator: "Ninja",
			Path:      path.Join(sourceDirectoryPath, cmakeSubpath),
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
}

func CMakeConfigureFixPkgconfigPrefix(pkgconfigSubpath, cmakeSubpath string, options ...*runners.CMakeOptions) func(*StandardBuilder, string, string) error {
	return func(sb *StandardBuilder, sourceDirectoryPath, buildDirectoryPath string) error {
		err := CMakeConfigure(cmakeSubpath, options...)(sb, sourceDirectoryPath, buildDirectoryPath)
		if err != nil {
			return trace.Wrap(err, "failed to run CMake for %s", sb.Name)
		}

		err = updatePkgconfigPrefix(path.Join(buildDirectoryPath, pkgconfigSubpath))
		if err != nil {
			return trace.Wrap(err, "failed to update pkgconfig prefix for %q", sb.Name)
		}

		return nil
	}
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

func NinjaBuild(buildTargets ...string) func(*StandardBuilder, string) error {
	if len(buildTargets) == 0 {
		buildTargets = append(buildTargets, "install")
	}

	return func(sb *StandardBuilder, buildDirectoryPath string) error {
		_, err := runners.Run(runners.CommandRunner{
			Command:   "ninja",
			Arguments: buildTargets,
			GenericRunner: runners.GenericRunner{
				WorkingDirectory:     buildDirectoryPath,
				EnvironmentVariables: sb.GetEnvironmentVariables(),
			},
		})
		if err != nil {
			return trace.Wrap(err, "failed to build %s", sb.Name)
		}

		return nil
	}
}
