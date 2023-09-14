package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

const defaultRemoteName string = "origin"

type GitRepo struct {
	*Source
	RepoUrl string // Fully qualified Git repo URL, e.g. https://github.com/some-user/some-repo or git://sourceware.org/git/binutils-gdb.git
	Ref     string // Fully qualified Git reference, e.g. "HEAD" or "refs/heads/master"
}

func NewGitRepo(repoDirectory, repoUrl, ref string) *GitRepo {
	return &GitRepo{
		Source:  NewSource(repoDirectory),
		RepoUrl: repoUrl,
		Ref:     ref,
	}
}

func (gr *GitRepo) Download(ctx context.Context) error {
	err := gr.Setup()
	if err != nil {
		return trace.Wrap(err, "failed to perform source setup for git repo %q", gr.PrettyName())
	}

	downloadDirectoryPath := gr.FullDownloadPath()
	repo, remoteName, err := gr.getOrCreateRepo(downloadDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to get repo")
	}

	// // TODO remove
	// cloneOptions := git.CloneOptions{
	// 	URL:               gr.RepoUrl,
	// 	ReferenceName:     plumbing.ReferenceName(gr.Ref),
	// 	SingleBranch:      true,
	// 	RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	// 	Tags:              git.NoTags,
	// 	Progress:          os.Stdout, // TODO revert or pipe to logger
	// 	Depth:             1,
	// }
	// _, err = git.PlainCloneContext(ctx, downloadDirectoryPath, false, &cloneOptions)
	// if err != nil {
	// 	return trace.Wrap(err, "failed to clone repo %q to %q", gr.PrettyName(), downloadDirectoryPath)
	// }

	err = gr.cloneInitializedRepo(repo, remoteName)
	if err != nil {
		return trace.Wrap(err, "failed to clone repo from remote %q to path %q", remoteName, downloadDirectoryPath)
	}

	return nil
}

func (gr *GitRepo) getOrCreateRepo(repoPath string) (*git.Repository, string, error) {
	shouldCheckForPreExistingRepo, err := gr.couldDirectoryContainPreExistingRepo(repoPath)
	if err != nil {
		return nil, "", trace.Wrap(err, "failed to check if %q could contain a pre-existing repo", repoPath)
	}

	// Check if the download directory contains a git repo already
	if shouldCheckForPreExistingRepo {
		repo, remoteName, err := gr.getExistingRepo(repoPath)
		if err != nil {
			return nil, "", trace.Wrap(err, "failed to get pre-existing repo at path %q", repoPath)
		}

		return repo, remoteName, nil
	}

	repo, remoteName, err := gr.createNewRepo(repoPath)
	if err != nil {
		return nil, "", trace.Wrap(err, "failed to create remote repo at path %q", repoPath)
	}

	return repo, remoteName, nil
}

func (gr *GitRepo) couldDirectoryContainPreExistingRepo(directoryPath string) (bool, error) {
	directoryHandle, err := os.Open(directoryPath)
	defer utils.Close(directoryHandle, &err)
	if err != nil {
		return false, trace.Wrap(err, "failed to open directory %q", directoryPath)
	}

	directoryContents, err := directoryHandle.Readdirnames(0)
	if err != nil {
		return false, trace.Wrap(err, "failed to list directory %q contents", directoryPath)
	}

	return len(directoryContents) > 0, nil
}

func (gr *GitRepo) getExistingRepo(repoPath string) (*git.Repository, string, error) {
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{DetectDotGit: false})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, "", trace.Wrap(err, "the directory %q contains contents but is not a git repository", repoPath)
		}
		return nil, "", trace.Wrap(err, "failed to open git repository at path %q", repoPath)
	}

	remoteName, err := gr.getRepoRemoteName(repo)
	if err != nil {
		return nil, "", trace.Wrap(err, "failed to get remote with fetch URL of %q for pre-existing repo at %q", gr.RepoUrl, repoPath)
	}

	return repo, remoteName, nil
}

func (gr *GitRepo) getRepoRemoteName(repo *git.Repository) (string, error) {
	repoRemotes, err := repo.Remotes()
	if err != nil {
		return "", trace.Wrap(err, "failed to get list of remotes for repository")
	}

	var remoteName string
	doesRepoContainMatchingRemote := slices.ContainsFunc(repoRemotes, func(remote *git.Remote) bool {
		matchedRemoteName := gr.getRemoteNameForFetchURL(remote)
		if matchedRemoteName != "" {
			remoteName = matchedRemoteName
			return true
		}

		return false
	})

	if !doesRepoContainMatchingRemote {
		return "", trace.Errorf("found git repo but it doesn't contain a remote with URL %q", gr.RepoUrl)
	}

	return remoteName, nil
}

// Returns an empty string if the fetch URL does not match the repo URL
// TODO rename this to something more meaningful
func (gr *GitRepo) getRemoteNameForFetchURL(remote *git.Remote) string {
	remoteConfig := remote.Config()
	remoteURLs := remoteConfig.URLs
	if len(remoteURLs) == 0 {
		return ""
	}

	// The fetch URL will always be the first one
	if remoteURLs[0] != gr.RepoUrl {
		return ""
	}

	return remoteConfig.Name
}

func (gr *GitRepo) createNewRepo(repoPath string) (*git.Repository, string, error) {
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		return nil, "", trace.Wrap(err, "failed to initialize git repository at %q", repoPath)
	}

	refSpec, err := gr.getRefspecForReference(defaultRemoteName)
	if err != nil {
		return nil, "", trace.Wrap(err, "failed to create refspec")
	}

	// TODO determine if setting the refspec here is beneficial
	_, err = repo.CreateRemote(&config.RemoteConfig{Name: defaultRemoteName, URLs: []string{gr.RepoUrl}, Fetch: []config.RefSpec{refSpec}})
	if err != nil {
		return nil, "", trace.Wrap(err, "failed to create remote for repository at %q", repoPath)
	}

	return repo, defaultRemoteName, nil
}

func (gr *GitRepo) cloneInitializedRepo(repo *git.Repository, remoteName string) error {
	refSpec, err := gr.getRefspecForReference(remoteName)
	if err != nil {
		return trace.Wrap(err, "failed to create refspec")
	}

	// Fetch the ref from the remote
	err = repo.Fetch(&git.FetchOptions{RemoteName: remoteName, Depth: 1, Tags: git.NoTags, RefSpecs: []config.RefSpec{refSpec}})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return trace.Wrap(err, "failed to fetch ref from remote %q", remoteName)
	}

	repoWorktree, err := repo.Worktree()
	if err != nil {
		return trace.Wrap(err, "failed to get repo worktree for repo")
	}

	err = gr.checkoutRef(repo, repoWorktree)
	if err != nil {
		return trace.Wrap(err, "failed to checkout ref from repo")
	}

	err = gr.updateSubmodules(repoWorktree)
	if err != nil {
		return trace.Wrap(err, "failed to update submodules for repo")
	}

	return nil
}

func (gr *GitRepo) getRefspecForReference(remoteName string) (config.RefSpec, error) {
	reference := plumbing.ReferenceName(gr.Ref)
	if reference.IsTag() {
		return config.RefSpec(fmt.Sprintf("+refs/tags/%s:refs/tags/%[1]s", reference.Short())), nil
	}

	if !reference.IsBranch() {
		return "", trace.Errorf("unsupported ref %q", reference)

	}

	if reference == plumbing.HEAD {
		return config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remoteName)), nil
	}
	return config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%[1]s", reference.Short(), remoteName)), nil
}

func (gr *GitRepo) checkoutRef(repo *git.Repository, repoWorktree *git.Worktree) error {
	reference := plumbing.ReferenceName(gr.Ref)
	commitHash, err := gr.getCommitHash(repo)
	if err != nil {
		return trace.Wrap(err, "failed to get commit hash for reference")
	}

	checkoutOptions := &git.CheckoutOptions{
		Force: true,
	}
	if reference.IsBranch() {
		checkoutOptions.Branch = reference
	} else {
		checkoutOptions.Hash = commitHash
	}

	err = repoWorktree.Checkout(checkoutOptions)
	if err != nil {
		return trace.Wrap(err, "failed to checkout ref %q for repo", reference)
	}

	return nil
}

// Gets the commit reference associated with `gr.Ref`, even if it does not directly point to
// a commit (such as annotated tags)
func (gr *GitRepo) getCommitHash(repo *git.Repository) (plumbing.Hash, error) {
	referenceName := plumbing.ReferenceName(gr.Ref)
	if !referenceName.IsTag() {
		reference, err := repo.Reference(referenceName, true)
		if err != nil {
			return plumbing.ZeroHash, trace.Wrap(err, "failed to get commit reference for %q", referenceName)
		}

		return reference.Hash(), nil
	}

	tagReference, err := repo.Tag(referenceName.Short())
	if err != nil {
		return plumbing.ZeroHash, trace.Wrap(err, "failed to get tag reference for %q", referenceName)
	}

	commitHash, err := gr.getCommitHashForTagHash(tagReference.Hash(), repo)
	if err != nil {
		return plumbing.ZeroHash, trace.Wrap(err, "failed to get commit hash for tag %q", referenceName)
	}

	return commitHash, nil
}

func (gr *GitRepo) getCommitHashForTagHash(tagHash plumbing.Hash, repo *git.Repository) (plumbing.Hash, error) {
	tagObject, err := repo.TagObject(tagHash)
	switch err {
	case plumbing.ErrObjectNotFound:
		// The tag is a lightweight tag, which does not need to be resolved further
		return tagObject.Target, nil
	case nil:
		// Tag is an annotated tag, which has an object separate from the commit
		commitHash, err := gr.getCommitHashForTagObject(*tagObject, repo)
		if err != nil {
			return plumbing.ZeroHash, trace.Wrap(err, "failed to get commit hash for tag object %q", tagObject.Name)
		}

		return commitHash, nil
	default:
		return plumbing.ZeroHash, trace.Wrap(err, "failed to resolve reference hash %q to tag object", tagHash)
	}
}

func (gr *GitRepo) getCommitHashForTagObject(tagObject object.Tag, repo *git.Repository) (plumbing.Hash, error) {
	for {
		switch tagObject.TargetType {
		case plumbing.CommitObject:
			commit, err := repo.CommitObject(tagObject.Target)
			if err != nil {
				return plumbing.ZeroHash, trace.Wrap(err, "failed to get targeted commit for tag object %q", tagObject.Name)
			}

			return commit.Hash, nil
		case plumbing.TagObject:
			targetTagObject, err := repo.TagObject(tagObject.Target)
			if err != nil {
				return plumbing.ZeroHash, trace.Wrap(err, "failed to get targeted tag object for tag object %q", tagObject.Name)
			}
			tagObject = *targetTagObject
			continue
		default:
			return plumbing.ZeroHash, trace.Errorf("unsupported tag object target type %q", tagObject.TargetType)
		}
	}
}

func (gr *GitRepo) updateSubmodules(repoWorktree *git.Worktree) error {
	repoSubmodules, err := repoWorktree.Submodules()
	if err != nil {
		return trace.Wrap(err, "failed to get submodules for repo")
	}

	err = repoSubmodules.Update(&git.SubmoduleUpdateOptions{Init: true, RecurseSubmodules: git.DefaultSubmoduleRecursionDepth, Depth: 1})
	if err != nil {
		return trace.Wrap(err, "failed to update submodules for repo")
	}

	return nil
}

func (gr *GitRepo) PrettyName() string {
	return fmt.Sprintf("%s@%s", gr.RepoUrl, gr.Ref)
}
