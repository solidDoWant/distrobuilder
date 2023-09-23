package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultXZRef string = "refs/tags/v5.4.4"
	XZRepoUrl    string = "https://github.com/tukaani-project/xz.git"
)

func NewXZGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultXZRef
	}

	return source.NewGitRepo("xz", XZRepoUrl, ref)
}
