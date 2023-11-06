package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultBzip2Ref string = "66c46b8c9436613fd81bc5d03f63a61933a4dcc3" // Master branch commit with CMake support
	Bzip2RepoUrl    string = "https://gitlab.com/bzip2/bzip2.git"
)

func NewBzip2GitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultBzip2Ref
	}

	return source.NewGitRepo("bzip2", Bzip2RepoUrl, ref)
}
