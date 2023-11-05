package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLibFUSERef string = "refs/tags/fuse-3.16.2"
	LibFUSERepoUrl    string = "https://github.com/libfuse/libfuse.git"
)

func NewLibFUSEGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultLibFUSERef
	}

	return source.NewGitRepo("LibFUSE", LibFUSERepoUrl, ref)
}
