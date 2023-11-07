package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultGDBMRef string = "refs/tags/v1.23"
	GDBMRepoUrl    string = "git://git.gnu.org.ua/gdbm.git"
)

func NewGDBMGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultGDBMRef
	}

	return source.NewGitRepo("gdbm", GDBMRepoUrl, ref)
}
