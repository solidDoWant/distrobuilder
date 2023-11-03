package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultDejaVuFontsRef string = "refs/tags/version_2_37"
	DejaVuFontsRepoUrl    string = "https://github.com/dejavu-fonts/dejavu-fonts.git"
)

func NewDejaVuFontsGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultDejaVuFontsRef
	}

	return source.NewGitRepo("DejaVuFonts", DejaVuFontsRepoUrl, ref)
}
