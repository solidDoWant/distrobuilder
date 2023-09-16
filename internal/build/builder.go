package build

import (
	"context"
	"log/slog"
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/source"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type Builder interface {
	CheckHostRequirements() error
	Build(context.Context) error
	VerifyBuild(context.Context) error
	// RequiredSpace() int	// TODO
}

type SourceBuilder struct {
	SourceDirectoryPath string
}

type FilesystemOutputBuilder struct {
	OutputDirectoryPath string
}

type ToolchainRequiredBuilder struct {
	ToolchainPath string
}

func (trb *ToolchainRequiredBuilder) GetPathForTool(tool string) string {
	return path.Join(trb.ToolchainPath, tool)
}

func setupForBuild(ctx context.Context, repo *source.GitRepo, outputDirectoryPath string) (*utils.Directory, *utils.Directory, error) {
	repoReadableName := repo.String()
	sourceDirectory := repo.FullDownloadPath()

	slog.Info("Cloning git repo", "repo", repoReadableName, "download_path", sourceDirectory)
	err := repo.Download(ctx)
	if err != nil {
		return nil, nil, trace.Wrap(err, "failed to clone %q", repoReadableName)
	}

	slog.Info("Creating build and output directories")
	buildDirectory := utils.NewDirectory("")
	err = buildDirectory.Create()
	if err != nil {
		return nil, nil, trace.Wrap(err, "failed to create temporary build directory")
	}

	if outputDirectoryPath == "" {
		outputDirectoryPath = utils.GetTempDirectoryPath()
	}
	outputDirectory := utils.NewDirectory(outputDirectoryPath)
	err = outputDirectory.Create()
	if err != nil {
		return nil, nil, trace.Wrap(err, "failed to create output directory at %q", outputDirectoryPath)
	}
	slog.Debug("Created build and output directories", "build_directory", buildDirectory.Path, "output_directory", outputDirectoryPath)

	return buildDirectory, outputDirectory, nil
}
