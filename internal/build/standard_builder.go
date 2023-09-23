package build

import (
	"context"
	"fmt"
	"log/slog"
	"path"

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
