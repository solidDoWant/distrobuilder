package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLZ4Ref string = "refs/tags/v1.9.4"
	LZ4RepoUrl    string = "https://github.com/lz4/lz4.git"
)

func NewLZ4GitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultLZ4Ref
	}

	return source.NewGitRepo("lz4", LZ4RepoUrl, ref)
}
