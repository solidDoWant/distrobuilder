package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type Zstd struct {
	StandardBuilder
}

func NewZstd() *Zstd {
	return &Zstd{
		StandardBuilder: StandardBuilder{
			Name:    "zstd",
			GitRepo: git_source.NewZstdGitRepo,
			DoConfiguration: CMakeConfigureFixPkgconfigPrefix(
				path.Join("lib", "libzstd.pc"),
				path.Join("build", "cmake"),
				&runners.CMakeOptions{
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
				},
			),
			DoBuild: NinjaBuild(),
			BinariesToCheck: []string{
				path.Join("usr", "bin", "zstd"),
				path.Join("usr", "bin", "unzstd"),
				path.Join("usr", "lib", "libzstd.so"),
			},
		},
	}
}
