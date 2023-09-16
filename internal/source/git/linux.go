package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLinuxRef string = "refs/tags/v6.5"
	LinuxRepoUrl    string = "git://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git"
)

func NewLinuxGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultLinuxRef
	}

	return source.NewGitRepo("linux", LinuxRepoUrl, ref)
}
