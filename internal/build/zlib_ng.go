package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type ZlibNg struct {
	StandardBuilder
}

func NewZLibNg() *ZlibNg {
	instance := &ZlibNg{
		StandardBuilder: StandardBuilder{
			Name: "zlib-ng",
			BinariesToCheck: []string{
				path.Join("usr", "lib", "libz.so"),
				path.Join("usr", "bin", "minigzip"),
				path.Join("usr", "bin", "minideflate"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (zng *ZlibNg) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewZlibNgGitRepo(repoDirectoryPath, ref)
}

func (zng *ZlibNg) DoConfiguration(buildDirectoryPath string) error {
	cmakeOptions := &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"ZLIB_COMPAT":   args.OnValue(),
			"INSTALL_UTILS": args.OnValue(),
		},
	}
	return zng.CMakeConfigure(buildDirectoryPath, cmakeOptions)
}

func (zng *ZlibNg) DoBuild(buildDirectoryPath string) error {
	return zng.NinjaBuild(buildDirectoryPath)
}
