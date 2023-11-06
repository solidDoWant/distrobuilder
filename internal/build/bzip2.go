package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type Bzip2 struct {
	StandardBuilder
}

func NewBzip2() *Bzip2 {
	instance := &Bzip2{
		StandardBuilder: StandardBuilder{
			Name: "bzip2",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "bzip2"),
				path.Join("usr", "bin", "bzgrep"),
				path.Join("usr", "bin", "bzip2recover"),
				path.Join("usr", "bin", "bzmore"),
				path.Join("usr", "lib", "libbz2.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (z *Bzip2) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewBzip2GitRepo(repoDirectoryPath, ref)
}

func (z *Bzip2) DoConfiguration(buildDirectoryPath string) error {
	cmakeOptions := &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"ENABLE_EXAMPLES":          args.OffValue(),
			"ENABLE_APP":               args.OnValue(),
			"ENABLE_STATIC_LIB":        args.OnValue(),
			"ENABLE_SHARED_LIB":        args.OnValue(),
			"ENABLE_STATIC_LIB_IS_PIC": args.OnValue(),
		},
	}
	return z.CMakeConfigure(buildDirectoryPath, cmakeOptions)
}

func (z *Bzip2) DoBuild(buildDirectoryPath string) error {
	return z.NinjaBuild(buildDirectoryPath)
}
