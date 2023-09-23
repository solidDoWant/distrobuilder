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
	cp "github.com/otiai10/copy"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type XZ struct {
	StandardBuilder
}

func NewXZ() *XZ {
	return &XZ{
		StandardBuilder: StandardBuilder{
			Name:    "xz",
			GitRepo: git_source.NewXZGitRepo,
			BinariesToCheck: []string{
				path.Join("usr", "bin", "xz"),
				path.Join("usr", "bin", "xzdec"),
				path.Join("usr", "lib", "liblzma.so"),
			},
		},
	}
}

// XZ has a relatively complicated build process that requires a two stage build
func (xz *XZ) Build(ctx context.Context) error {
	slog.Info(fmt.Sprintf("Starting %s build", xz.Name))
	repo := xz.GitRepo(xz.SourceDirectoryPath, xz.GitRef)
	sourcePath := repo.FullDownloadPath()

	// This is the final output directory of both builds stages
	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, xz.OutputDirectoryPath)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup for %s build", xz.Name)
	}
	xz.OutputDirectoryPath = outputDirectory.Path

	err = xz.runAutogen(sourcePath, buildDirectory.Path)
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

	err = xz.runMake(buildDirectory.Path, outputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to execute makefile targets")
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

	err = xz.runMake(path.Join(buildDirectory.Path, "src", "liblzma"), outputDirectoryPath, "all")
	if err != nil {
		return trace.Wrap(err, "failed to build static liblzma")
	}

	err = xz.runMake(path.Join(buildDirectory.Path, "src", "xzdec"), outputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to build static xzdec")
	}

	return nil
}

func (xz *XZ) configureStage1(sourceDirectoryPath, buildDirectoryPath string) error {
	return xz.runConfigure(sourceDirectoryPath, buildDirectoryPath, []string{
		"--disable-static",
		"--disable-xzdec",
		"--disable-lzmadec",
	})
}

func (xz *XZ) configureStage2(sourceDirectoryPath, buildDirectoryPath string) error {
	return xz.runConfigure(sourceDirectoryPath, buildDirectoryPath, []string{
		"--disable-shared",
		"--disable-nls",
		"--disable-encoders",
		"--disable-threads",
	})
}

func (xz *XZ) runAutogen(sourceDirectoryPath, buildDirectoryPath string) error {
	// Autogen is somewhat strange and does not support generating files in
	// a separate build directory. To prevent contaminating the source folder,
	// the source contents are first copied to the build directory.
	// Somehow Go does not have a builtin library for copying directories, so
	// use this third party one that should cover most corner cases.
	err := cp.Copy(sourceDirectoryPath, buildDirectoryPath, cp.Options{
		// Skip copying ".git*" files, such as the ".git" directory and ".gitignore"
		Skip: func(srcInfo os.FileInfo, src, dest string) (bool, error) {
			return strings.HasPrefix(srcInfo.Name(), ".git"), nil
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to copy source directory %q contents to build directory %q", sourceDirectoryPath, buildDirectoryPath)
	}

	_, err = runners.Run(&runners.CommandRunner{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory:     buildDirectoryPath,
			EnvironmentVariables: xz.GetEnvironmentVariables(),
		},
		Command: path.Join(buildDirectoryPath, "autogen.sh"),
	})
	if err != nil {
		return trace.Wrap(err, "command autogen.sh failed in build directory %q", buildDirectoryPath)
	}

	return nil
}

func (xz *XZ) runConfigure(sourceDirectoryPath, buildDirectoryPath string, flags []string) error {
	_, err := runners.Run(&runners.Configure{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory:     buildDirectoryPath,
			EnvironmentVariables: xz.GetEnvironmentVariables(),
		},
		Options: []*runners.ConfigureOptions{
			xz.ToolchainRequiredBuilder.GetConfigurenOptions(""),
			xz.RootFSBuilder.GetConfigurenOptions(),
			{
				AdditionalArgs: map[string]args.IValue{
					"--prefix": args.StringValue("/"), // Path is relative to DESTDIR, set when invoking make
					"--srcdir": args.StringValue(sourceDirectoryPath),
					"--host":   args.StringValue(xz.ToolchainRequiredBuilder.Triplet.String()),
				},
				AdditionalFlags: pie.Map(flags, func(flag string) args.IValue { return args.StringValue(flag) }),
			},
		},
		ConfigurePath: path.Join(sourceDirectoryPath, "configure"),
	})

	return err
}

func (xz *XZ) runMake(makefileDirectoryPath, outputDirectoryPath string, targets ...string) error {
	if len(targets) == 0 {
		targets = append(targets, "all", "install-strip")
	}

	_, err := runners.Run(&runners.Make{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory:     makefileDirectoryPath,
			EnvironmentVariables: xz.GetEnvironmentVariables(),
		},
		Path:    ".",
		Targets: targets,
		Variables: map[string]string{
			"DESTDIR": path.Join(outputDirectoryPath, "usr"), // This must be set so that all files are installed/written to the output directory
		},
	})

	if err != nil {
		return trace.Wrap(err, "musl libc make build failed")
	}

	return nil
}
