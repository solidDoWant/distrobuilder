package build

import (
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type LibFUSE struct {
	StandardBuilder
}

func NewLibFUSE() *LibFUSE {
	instance := &LibFUSE{
		StandardBuilder: StandardBuilder{
			Name: "libfuse",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "fusermount3"),
				path.Join("usr", "sbin", "mount.fuse3"),
				path.Join("usr", "lib", "libfuse3.so"),
			},
		},
	}

	// There is not currently Golang syntactic sugar for this pattern
	instance.IStandardBuilder = instance
	return instance
}

func (lfuse *LibFUSE) GetGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	return git_source.NewLibFUSEGitRepo(repoDirectoryPath, ref)
}

func (lfuse *LibFUSE) DoConfiguration(buildDirectoryPath string) error {
	mesonOptions := &runners.MesonOptions{
		Options: map[string]args.IValue{
			"examples": args.StringValue("false"),
			"tests":    args.StringValue("false"),
		},
	}
	return lfuse.MesonSetup(buildDirectoryPath, mesonOptions)
}

func (lfuse *LibFUSE) DoBuild(buildDirectoryPath string) error {
	err := lfuse.NinjaBuild(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to build libfuse with Ninja")
	}

	err = utils.CreateSymlink(path.Join(lfuse.OutputDirectoryPath, "usr", "bin", "fusermount"), "fusermount3")
	if err != nil {
		return trace.Wrap(err, "failed to create fusermount symlink")
	}

	err = utils.CreateSymlink(path.Join(lfuse.OutputDirectoryPath, "usr", "sbin", "mount.fuse"), "mount.fuse3")
	if err != nil {
		return trace.Wrap(err, "failed to create fusermount symlink")
	}

	return nil
}
