package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultFreeTypeRef string = "refs/tags/VER-2-13-2"
	FreeTypeRepoUrl    string = "https://gitlab.freedesktop.org/freetype/freetype.git"
)

func NewFreeTypeGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultFreeTypeRef
	}

	return source.NewGitRepo("FreeType", FreeTypeRepoUrl, ref)
}
