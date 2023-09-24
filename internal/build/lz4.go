package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type LZ4 struct {
	StandardBuilder
}

func NewLZ4() *LZ4 {
	return &LZ4{
		StandardBuilder: StandardBuilder{
			Name:    "LZ4",
			GitRepo: git_source.NewLZ4GitRepo,
			DoConfiguration: CMakeConfigureFixPkgconfigPrefix(
				"liblz4.pc",
				path.Join("build", "cmake"),
				&runners.CMakeOptions{
					Defines: map[string]args.IValue{
						"BUILD_SHARED_LIBS": args.OnValue(),
						"BUILD_STATIC_LIBS": args.OnValue(),
					},
				}),
			DoBuild: NinjaBuild(),
			BinariesToCheck: []string{
				path.Join("usr", "bin", "lz4"),
				path.Join("usr", "lib", "liblz4.so"),
			},
		},
	}
}
