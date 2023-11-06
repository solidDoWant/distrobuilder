package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type FreeType struct {
	StandardBuilder
}

func NewFreeType() *FreeType {
	instance := &FreeType{
		StandardBuilder: StandardBuilder{
			Name: "FreeType",
			BinariesToCheck: []string{
				path.Join("usr", "lib", "libfreetype.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (z *FreeType) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewFreeTypeGitRepo(repoDirectoryPath, ref)
}

func (z *FreeType) DoConfiguration(buildDirectoryPath string) error {
	cmakeOptions := &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"BUILD_SHARED_LIBS": args.OnValue(),
			"FT_REQUIRE_ZLIB":   args.OnValue(),
			// TODO
			// "FT_REQUIRE_PNG": args.OnValue(),
			"FT_REQUIRE_BZIP2": args.OnValue(),
			// "FT_REQUIRE_BROTLI": args.OnValue(),
			// "FT_REQUIRE_HARFBUZZ": args.OnValue(),
		},
	}
	return z.CMakeConfigure(buildDirectoryPath, cmakeOptions)
}

func (z *FreeType) DoBuild(buildDirectoryPath string) error {
	return z.NinjaBuild(buildDirectoryPath)
}
