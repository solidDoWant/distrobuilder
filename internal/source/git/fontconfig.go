package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultFontConfigRef string = "refs/tags/2.14.2"
	FontConfigRepoUrl    string = "git://cgit.freedesktop.org/fontconfig"
)

func NewFontConfigGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultFontConfigRef
	}

	return source.NewGitRepo("FontConfig", FontConfigRepoUrl, ref)
}
