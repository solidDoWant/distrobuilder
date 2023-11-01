package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLibreSSLRef string = "refs/tags/v3.8.1"
	LibreSSLRepoUrl    string = "https://github.com/libressl/portable.git"
)

func NewLibreSSLGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultLibreSSLRef
	}

	return source.NewGitRepo("LibreSSL", LibreSSLRepoUrl, ref)
}
