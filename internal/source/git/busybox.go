package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultBusyBoxRef string = "refs/tags/1_36_1"
	BusyBoxRepoUrl    string = "git://git.busybox.net/busybox"
)

func NewBusyBoxGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultBusyBoxRef
	}

	return source.NewGitRepo("busybox", BusyBoxRepoUrl, ref)
}
