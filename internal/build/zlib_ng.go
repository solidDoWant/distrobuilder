package build

import (
	"context"
	"log/slog"
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type ZlibNg struct {
	SourceBuilder
	FilesystemOutputBuilder
	ToolchainRequiredBuilder
	GitRefBuilder
	RootFSBuilder
}

func (zn *ZlibNg) CheckHostRequirements() error {
	err := zn.CheckToolsExist()
	if err != nil {
		return trace.Wrap(err, "failed to verify that all required toolchain tools exist")
	}

	return nil
}

func (zn *ZlibNg) Build(ctx context.Context) error {
	slog.Info("Starting zlib-ng build")
	repo := git_source.NewZlibNgGitRepo(zn.SourceDirectoryPath, zn.GitRef)
	sourcePath := repo.FullDownloadPath()

	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, zn.OutputDirectoryPath)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup for zlib-ng build")
	}
	zn.OutputDirectoryPath = outputDirectory.Path

	err = zn.runZlibNgCMake(sourcePath, buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to configure zlib-ng")
	}

	err = zn.runZlibNgBuild(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to build zlib-ng")
	}

	return nil
}

func (zn *ZlibNg) runZlibNgCMake(sourceDirectoryPath, buildDirectoryPath string) error {
	_, err := runners.Run(&runners.CMake{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectoryPath,
		},
		Generator: "Ninja",
		Path:      sourceDirectoryPath,
		Options: []*runners.CMakeOptions{
			zn.FilesystemOutputBuilder.GetCMakeOptions("usr"),
			zn.ToolchainRequiredBuilder.GetCMakeOptions(),
			zn.RootFSBuilder.GetCMakeOptions(),
			{
				Defines: map[string]args.IValue{
					"ZLIB_COMPAT":   args.OnValue(),
					"INSTALL_UTILS": args.OnValue(),
				},
			},
		},
	})

	if err != nil {
		return trace.Wrap(err, "failed to create generator file for zlib-ng")
	}

	return nil
}

func (zn *ZlibNg) runZlibNgBuild(buildDirectoryPath string) error {
	_, err := runners.Run(runners.CommandRunner{
		Command:   "/workspaces/distrobuilder/test.sh",
		Arguments: []string{"ninja", "install"},
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectoryPath,
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to build zlib-ng")
	}

	return nil
}

func (zn *ZlibNg) VerifyBuild(ctx context.Context) error {
	filesToCheck := []string{
		path.Join(zn.OutputDirectoryPath, "usr", "lib", "libz.so"),
		path.Join(zn.OutputDirectoryPath, "usr", "bin", "minigzip"),
		path.Join(zn.OutputDirectoryPath, "usr", "bin", "minideflate"),
	}

	for _, filePath := range filesToCheck {
		err := zn.VerifyTargetElfFile(filePath)
		if err != nil {
			return trace.Wrap(err, "built file %q did not match the expected ELF values", filePath)
		}
	}

	return nil
}
