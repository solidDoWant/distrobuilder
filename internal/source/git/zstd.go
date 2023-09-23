package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultZstdRef string = "refs/tags/v1.5.5"
	ZstdRepoUrl    string = "https://github.com/facebook/zstd.git"
)

func NewZstdGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultZstdRef
	}

	return source.NewGitRepo("zstd", ZstdRepoUrl, ref)
}
