package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultMuslFTSRef string = "refs/tags/v1.2.7"
	MuslFTSRepoUrl    string = "https://github.com/void-linux/musl-fts.git"
)

func NewMuslFTSGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultMuslFTSRef
	}

	return source.NewGitRepo("musl-fts", MuslFTSRepoUrl, ref)
}
