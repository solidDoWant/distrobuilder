package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLibtoolRef string = "refs/tags/v2.4.7"
	LibtoolRepoUrl    string = "git://git.savannah.gnu.org/libtool.git"
)

func NewLibtoolGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultLibtoolRef
	}

	return source.NewGitRepo("libtool", LibtoolRepoUrl, ref)
}
