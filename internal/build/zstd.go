package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type Zstd struct {
	StandardBuilder
}

func NewZstd() *Zstd {
	instance := &Zstd{
		StandardBuilder: StandardBuilder{
			Name: "zstd",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "zstd"),
				path.Join("usr", "bin", "unzstd"),
				path.Join("usr", "lib", "libzstd.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (z *Zstd) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewZstdGitRepo(repoDirectoryPath, ref)
}

func (z *Zstd) DoConfiguration(buildDirectoryPath string) error {
	cmakeOptions := &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"ZSTD_MULTITHREAD_SUPPORT":  args.OnValue(),
			"ZSTD_BUILD_SHARED":         args.OnValue(),
			"ZSTD_PROGRAMS_LINK_SHARED": args.OnValue(),
			"ZSTD_BUILD_STATIC":         args.OnValue(),
			"ZSTD_BUILD_TESTS":          args.OnValue(),
			"ZSTD_ZLIB_SUPPORT":         args.OnValue(),
			"ZSTD_LZMA_SUPPORT":         args.OnValue(),
			"ZSTD_LZ4_SUPPORT":          args.OnValue(),
		},
	}
	return z.CMakeConfigureWithPath(buildDirectoryPath, path.Join(z.SourceDirectoryPath, "build", "cmake"), cmakeOptions)
}

func (z *Zstd) DoBuild(buildDirectoryPath string) error {
	return z.NinjaBuild(buildDirectoryPath)
}
