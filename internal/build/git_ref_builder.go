package build

type IGitRefBuilder interface {
	SetGitRef(string)
	GetGitRef() string
}

type GitRefBuilder struct {
	GitRef string
}

func (grb *GitRefBuilder) SetGitRef(gitRef string) {
	grb.GitRef = gitRef
}

func (grb *GitRefBuilder) GetGitRef() string {
	return grb.GitRef
}
