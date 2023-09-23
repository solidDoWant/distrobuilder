package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type ZlibNg struct {
	StandardBuilder
}

func NewZLibNg() *ZlibNg {
	return &ZlibNg{
		StandardBuilder: StandardBuilder{
			Name:    "zlib-ng",
			GitRepo: git_source.NewZlibNgGitRepo,
			DoConfiguration: CMakeConfigure(
				"",
				&runners.CMakeOptions{
					Defines: map[string]args.IValue{
						"ZLIB_COMPAT":   args.OnValue(),
						"INSTALL_UTILS": args.OnValue(),
					},
				},
			),
			DoBuild: NinjaBuild(),
			BinariesToCheck: []string{
				path.Join("usr", "lib", "libz.so"),
				path.Join("usr", "bin", "minigzip"),
				path.Join("usr", "bin", "minideflate"),
			},
		},
	}
}
