package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultZlibNgRef string = "refs/tags/2.1.3"
	ZlibNgRepoUrl    string = "https://github.com/zlib-ng/zlib-ng.git"
)

func NewZlibNgGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultZlibNgRef
	}

	return source.NewGitRepo("zlib-ng", ZlibNgRepoUrl, ref)
}
