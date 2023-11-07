package build

import (
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type GDBM struct {
	StandardBuilder
}

func NewGDBM() *GDBM {
	instance := &GDBM{
		StandardBuilder: StandardBuilder{
			Name: "gdbm",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "gdbm_dump"),
				path.Join("usr", "bin", "gdbm_load"),
				path.Join("usr", "bin", "gdbmtool"),
				path.Join("usr", "lib", "libgdbm.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (gdbm *GDBM) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewGDBMGitRepo(repoDirectoryPath, ref)
}

func (gdbm *GDBM) DoConfiguration(buildDirectoryPath string) error {
	err := gdbm.Bootstrap(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to bootstrap %s", gdbm.Name)
	}

	err = gdbm.GNUConfigureWithSrc(buildDirectoryPath, buildDirectoryPath,
		"--enable-crash-tolerance",
	)
	if err != nil {
		return trace.Wrap(err, "failed to configure %s", gdbm.Name)
	}

	return nil
}

func (gdbm *GDBM) DoBuild(buildDirectoryPath string) error {
	err := gdbm.MakeBuild(buildDirectoryPath, gdbm.getMakeOptions(), "install")
	if err != nil {
		return trace.Wrap(err, "failed to perform make build on %s", gdbm.Name)
	}

	err = gdbm.RunLibtool(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run libtool on the build output")
	}

	return nil
}

func (gdbm *GDBM) getMakeOptions() []*runners.MakeOptions {
	return []*runners.MakeOptions{
		{
			Variables: map[string]args.IValue{
				"DESTDIR": args.StringValue(path.Join(gdbm.OutputDirectoryPath, "usr")),
			},
		},
	}
}
