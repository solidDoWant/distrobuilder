package build

import (
	"context"
	"fmt"
	"log/slog"
	"path"

	"github.com/gravitational/trace"
	cp "github.com/otiai10/copy"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type XZ struct {
	StandardBuilder
}

func NewXZ() *XZ {
	instance := &XZ{
		StandardBuilder: StandardBuilder{
			Name: "xz",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "xz"),
				path.Join("usr", "bin", "xzdec"),
				path.Join("usr", "lib", "liblzma.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (xz *XZ) GetGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	return git_source.NewXZGitRepo(repoDirectoryPath, ref)
}

func (xz *XZ) DoConfiguration(buildDirectoryPath string) error {
	// TODO this isn't great design, rework it
	panic("not implemented") // This is not needed because Build is overriden
}

func (xz *XZ) DoBuild(buildDirectoryPath string) error {
	panic("not implemented") // This is not needed because Build is overriden
}

// XZ has a relatively complicated build process that requires a two stage build
func (xz *XZ) Build(ctx context.Context) error {
	slog.Info(fmt.Sprintf("Starting %s build", xz.Name))
	repo := xz.GetGitRepo(xz.SourceDirectoryPath, xz.GitRef)
	sourcePath := repo.FullDownloadPath()

	// This is the final output directory of both builds stages
	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, xz.OutputDirectoryPath)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup for %s build", xz.Name)
	}
	xz.OutputDirectoryPath = outputDirectory.Path

	err = xz.Autogen(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to run autogen for %s build", xz.Name)
	}

	err = xz.buildStage1(buildDirectory.Path, outputDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to complete stage 1 %s build", xz.Name)
	}

	err = xz.buildStage2(buildDirectory.Path, outputDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to complete stage 2 %s build", xz.Name)
	}

	extraSourcePath := path.Join(sourcePath, "extra")
	extraDestinationPath := path.Join(outputDirectory.Path, "usr", "share", "doc", "xz", "extra")
	err = cp.Copy(extraSourcePath, extraDestinationPath, cp.Options{PreserveOwner: true, PreserveTimes: true})
	if err != nil {
		return trace.Wrap(err, "failed to copy extras from %q to output directory at %q", extraSourcePath, extraDestinationPath)
	}

	return nil
}

func (xz *XZ) buildStage1(sourceDirectoryPath, outputDirectoryPath string) error {
	buildDirectory, err := setupBuildDirectory()
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup build directory")
	}

	err = xz.configureStage1(sourceDirectoryPath, buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to execute configure in build directory %q", buildDirectory)
	}

	err = xz.MakeBuild(buildDirectory.Path, xz.getMakeOptions(), "all", "install-strip")
	if err != nil {
		return trace.Wrap(err, "failed to execute makefile targets")
	}

	err = updatePkgconfigPrefix(path.Join(outputDirectoryPath, "usr", "lib", "pkgconfig", "liblzma.pc"))
	if err != nil {
		return trace.Wrap(err, "failed to update pkgconfig prefix for %q", xz.Name)
	}

	return nil
}

func (xz *XZ) buildStage2(sourceDirectoryPath, outputDirectoryPath string) error {
	buildDirectory, err := setupBuildDirectory()
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup build directory")
	}

	err = xz.configureStage2(sourceDirectoryPath, buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to execute configure in build directory %q", buildDirectory)
	}

	makeOptions := xz.getMakeOptions()
	err = xz.MakeBuild(path.Join(buildDirectory.Path, "src", "liblzma"), makeOptions, "all")
	if err != nil {
		return trace.Wrap(err, "failed to build static liblzma")
	}

	err = xz.MakeBuild(path.Join(buildDirectory.Path, "src", "xzdec"), makeOptions, "all", "install-strip")
	if err != nil {
		return trace.Wrap(err, "failed to build static xzdec")
	}

	return nil
}

func (xz *XZ) configureStage1(sourceDirectoryPath, buildDirectoryPath string) error {
	return xz.GNUConfigure(sourceDirectoryPath, buildDirectoryPath,
		"--disable-static", "--disable-xzdec", "--disable-lzmadec")
}

func (xz *XZ) configureStage2(sourceDirectoryPath, buildDirectoryPath string) error {
	return xz.GNUConfigure(sourceDirectoryPath, buildDirectoryPath,
		"--disable-shared", "--disable-nls", "--disable-encoders", "--disable-threads")
}

func (xz *XZ) getMakeOptions() []*runners.MakeOptions {
	return []*runners.MakeOptions{
		{
			Variables: map[string]args.IValue{
				// This is set as an environment variable, but override it and specify the
				// usr subdirectory
				"DESTDIR": args.StringValue(path.Join(xz.OutputDirectoryPath, "usr")),
			},
		},
	}
}
