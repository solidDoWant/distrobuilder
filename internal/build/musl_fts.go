package build

import (
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type MuslFTS struct {
	StandardBuilder
}

func NewMuslFTS() *MuslFTS {
	instance := &MuslFTS{
		StandardBuilder: StandardBuilder{
			Name: "MuslFTS",
			BinariesToCheck: []string{
				path.Join("usr", "lib", "libfts.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (mfts *MuslFTS) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewMuslFTSGitRepo(repoDirectoryPath, ref)
}

func (mfts *MuslFTS) DoConfiguration(buildDirectoryPath string) error {
	err := mfts.Bootstrap(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to bootstrap %s", mfts.Name)
	}

	err = mfts.GNUConfigureWithSrc(buildDirectoryPath, buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to configure project")
	}

	return nil
}

func (mfts *MuslFTS) DoBuild(buildDirectoryPath string) error {
	err := mfts.MakeBuild(buildDirectoryPath, mfts.getMakeOptions(), "install")
	if err != nil {
		return trace.Wrap(err, "failed to perform make build")
	}

	err = mfts.UpdatePkgconfigsPrefixes(mfts.OutputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to update package config files")
	}

	err = mfts.RunLibtool(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run libtool on the build output")
	}

	return nil
}

func (mfts *MuslFTS) getMakeOptions() []*runners.MakeOptions {
	return []*runners.MakeOptions{
		{
			Variables: map[string]args.IValue{
				"DESTDIR": args.StringValue(path.Join(mfts.OutputDirectoryPath, "usr")),
			},
		},
	}
}
