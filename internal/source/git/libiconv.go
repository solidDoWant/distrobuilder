package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLibiconvRef string = "317dfadc6c68b3465205873b140200e5b0d0256f" // Needed for clang support on autogen
	LibiconvRepoUrl    string = "git://git.savannah.gnu.org/libiconv.git"
)

func NewLibiconvGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultLibiconvRef
	}

	return source.NewGitRepo("libiconv", LibiconvRepoUrl, ref)
}
