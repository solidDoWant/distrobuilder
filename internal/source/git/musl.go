package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultMuslRef string = "refs/tags/v1.2.4"
	MuslRepoUrl    string = "git://git.musl-libc.org/musl"
)

func NewMuslGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultMuslRef
	}

	return source.NewGitRepo("musl", MuslRepoUrl, ref)
}
