package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultPCRE2Ref string = "refs/tags/pcre2-10.42"
	PCRE2RepoUrl    string = "https://github.com/PCRE2Project/pcre2.git"
)

func NewPCRE2GitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultPCRE2Ref
	}

	return source.NewGitRepo("PCRE2", PCRE2RepoUrl, ref)
}
