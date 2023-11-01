package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type LZ4 struct {
	StandardBuilder
}

func NewLZ4Builder() *LZ4 {
	instance := &LZ4{
		StandardBuilder: StandardBuilder{
			Name: "LZ4",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "lz4"),
				path.Join("usr", "lib", "liblz4.so"),
			},
		},
	}

	// There is not currently Golang syntactic sugar for this pattern
	instance.IStandardBuilder = instance
	return instance
}

func (lz4 *LZ4) GetGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	return git_source.NewLZ4GitRepo(repoDirectoryPath, ref)
}

func (lz4 *LZ4) DoConfiguration(buildDirectoryPath string) error {
	cmakeOptions := &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"BUILD_SHARED_LIBS": args.OnValue(),
			"BUILD_STATIC_LIBS": args.OnValue(),
		},
	}
	return lz4.CMakeConfigureWithPath(buildDirectoryPath, path.Join(lz4.SourceDirectoryPath, "build", "cmake"), cmakeOptions)
}

func (lz4 *LZ4) DoBuild(buildDirectoryPath string) error {
	return lz4.NinjaBuild(buildDirectoryPath)
}
