package build

import (
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type Libtool struct {
	StandardBuilder
}

func NewLibtool() *Libtool {
	instance := &Libtool{
		StandardBuilder: StandardBuilder{
			Name:            "libtool",
			BinariesToCheck: []string{
				// path.Join("usr", "bin", "Libtool"),
				// path.Join("usr", "bin", "unLibtool"),
				// path.Join("usr", "lib", "libLibtool.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (l *Libtool) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewLibtoolGitRepo(repoDirectoryPath, ref)
}

func (l *Libtool) DoConfiguration(buildDirectoryPath string) error {
	err := l.Bootstrap(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to bootstrap %s", l.Name)
	}

	err = l.GNUConfigureWithSrc(buildDirectoryPath, buildDirectoryPath, "--enable-ltdl-install")
	if err != nil {
		return trace.Wrap(err, "failed to configure project")
	}

	return nil
}

func (l *Libtool) DoBuild(buildDirectoryPath string) error {
	err := l.MakeBuild(buildDirectoryPath, l.getMakeOptions(), "install")
	if err != nil {
		return trace.Wrap(err, "failed to perform make build")
	}

	// TODO figure out how to build the libtool script properly. The built libtool
	// uses a ton of vars from the build process, rather than the host system

	err = l.RunLibtool(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run libtool on the build output")
	}

	return nil
}

func (l *Libtool) getMakeOptions() []*runners.MakeOptions {
	return []*runners.MakeOptions{
		{
			Variables: map[string]args.IValue{
				// This is set as an environment variable, but override it and specify the
				// usr subdirectory
				"DESTDIR": args.StringValue(path.Join(l.OutputDirectoryPath, "usr")),
			},
		},
	}
}
