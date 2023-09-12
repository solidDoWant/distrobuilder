package source

import (
	"context"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gravitational/trace"
)

type GitRepo struct {
	*Source
	RepoUrl string // Fully qualified Git repo URL, e.g. https://github.com/some-user/some-repo or git://sourceware.org/git/binutils-gdb.git
	Ref     string // Fully qualified Git reference, e.g. "HEAD" or "refs/heads/master"
}

func NewGitRepo(downloadDirectory, repoUrl, ref string) *GitRepo {
	return &GitRepo{
		Source:  NewSource(downloadDirectory),
		RepoUrl: repoUrl,
		Ref:     ref,
	}
}

func (gr *GitRepo) Download(ctx context.Context) error {
	err := gr.Setup()
	if err != nil {
		return trace.Wrap(err, "failed to perform source setup for git repo %q", gr.PrettyName())
	}

	// Clone the repo, downloading only the target ref
	cloneOptions := git.CloneOptions{
		URL:               gr.RepoUrl,
		ReferenceName:     plumbing.ReferenceName(gr.Ref),
		SingleBranch:      true,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Tags:              git.NoTags,
		Progress:          os.Stdout, // TODO revert or pipe to logger
		Depth:             1,
	}
	_, err = git.PlainCloneContext(ctx, gr.DownloadPath, false, &cloneOptions)
	if err != nil {
		return trace.Wrap(err, "failed to clone repo %q to %q", gr.PrettyName(), gr.DownloadPath)
	}

	return nil
}

func (gr *GitRepo) PrettyName() string {
	return fmt.Sprintf("%s@%s", gr.RepoUrl, gr.Ref)
}
