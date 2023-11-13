package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

const defaultRemoteName string = "origin"

type GitRepo struct {
	*Source
	Name string // Name of the Git repo source
	Url  string // Fully qualified Git repo URL, e.g. https://github.com/some-user/some-repo or git://sourceware.org/git/binutils-gdb.git
	Ref  string // Fully qualified Git reference, e.g. "HEAD" or "refs/heads/master"
}

func NewGitRepo(name, url, ref string) *GitRepo {
	source := NewSource()
	source.DownloadPath = path.Join("git", name)

	return &GitRepo{
		Source: source,
		Name:   name,
		Url:    url,
		Ref:    ref,
	}
}

func (gr *GitRepo) Download(ctx context.Context) error {
	err := gr.Setup()
	if err != nil {
		return trace.Wrap(err, "failed to perform source setup for git repo %q", gr.String())
	}

	downloadDirectoryPath := gr.FullDownloadPath()
	repo, remoteName, err := gr.getOrCreateRepo(downloadDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to get repo")
	}

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
		return nil, "", trace.Wrap(err, "failed to get remote with fetch URL of %q for pre-existing repo at %q", gr.Url, repoPath)
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
		return "", trace.Errorf("found git repo but it doesn't contain a remote with URL %q", gr.Url)
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
	if remoteURLs[0] != gr.Url {
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
	_, err = repo.CreateRemote(&config.RemoteConfig{Name: defaultRemoteName, URLs: []string{gr.Url}, Fetch: []config.RefSpec{refSpec}})
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

	// Request all refspecs if an exact commit is requested, but the server does not support it
	if refSpec.IsExactSHA1() {
		isExactCommitSupported, err := gr.isExactSHA1Supported(repo, remoteName)
		if err != nil {
			return trace.Wrap(err, "failed to determine if exact commit is supported by server")
		}

		if !isExactCommitSupported {
			refSpec = "+refs/heads/*:refs/remotes/origin/*"
		}
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

func (gr *GitRepo) isExactSHA1Supported(repo *git.Repository, remoteName string) (bool, error) {
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return false, trace.Wrap(err, "failed to find remote named %q", remoteName)
	}

	// Fetch endpoint URL
	remoteFetchURL := remote.Config().URLs[0]
	endpoint, err := transport.NewEndpoint(remoteFetchURL)
	if err != nil {
		return false, trace.Wrap(err, "failed to get endpoint for remote %q", remoteName)
	}

	client, err := client.NewClient(endpoint)
	if err != nil {
		return false, trace.Wrap(err, "failed to create client for endpoint URL %q", remoteFetchURL)
	}

	session, err := client.NewUploadPackSession(endpoint, nil)
	defer utils.Close(session, &err)
	if err != nil {
		return false, trace.Wrap(err, "failed to create an upload pack session for %q", remoteFetchURL)
	}

	adversisedReferences, err := session.AdvertisedReferences()
	if err != nil {
		return false, trace.Wrap(err, "failed to get the advertised references for %q", remoteFetchURL)
	}

	return adversisedReferences.Capabilities.Supports(capability.AllowReachableSHA1InWant) &&
		adversisedReferences.Capabilities.Supports(capability.AllowTipSHA1InWant), nil
}

func (gr *GitRepo) getRefspecForReference(remoteName string) (config.RefSpec, error) {
	reference := plumbing.ReferenceName(gr.Ref)
	if reference.IsTag() {
		return config.RefSpec(fmt.Sprintf("+refs/tags/%s:refs/tags/%[1]s", reference.Short())), nil
	}

	if reference.IsBranch() {
		if reference == plumbing.HEAD {
			return config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remoteName)), nil
		}
		return config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%[1]s", reference.Short(), remoteName)), nil
	}

	if plumbing.IsHash(reference.String()) {
		refHash := reference.String()
		return config.RefSpec(fmt.Sprintf("+%s:refs/remotes/%s/%[1]s", refHash, remoteName)), nil
	}

	return "", trace.Errorf("unsupported ref %q", reference)
}

func (gr *GitRepo) checkoutRef(repo *git.Repository, repoWorktree *git.Worktree) error {
	checkoutOptions := &git.CheckoutOptions{
		Force: true,
	}

	reference := plumbing.ReferenceName(gr.Ref)
	if reference.IsBranch() {
		checkoutOptions.Branch = reference
	} else {
		commitHash, err := gr.getCommitHash(repo)
		if err != nil {
			return trace.Wrap(err, "failed to get commit hash for reference")
		}
		checkoutOptions.Hash = commitHash
	}

	err := repoWorktree.Checkout(checkoutOptions)
	if err != nil {
		return trace.Wrap(err, "failed to checkout ref %q for repo", reference)
	}

	return nil
}

// Gets the commit reference associated with `gr.Ref`, even if it does not directly point to
// a commit (such as annotated tags)
func (gr *GitRepo) getCommitHash(repo *git.Repository) (plumbing.Hash, error) {
	referenceName := plumbing.ReferenceName(gr.Ref)

	if plumbing.IsHash(gr.Ref) {
		return plumbing.NewHash(gr.Ref), nil
	}

	if referenceName.IsTag() {
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

	reference, err := repo.Reference(referenceName, true)
	if err != nil {
		return plumbing.ZeroHash, trace.Wrap(err, "failed to get commit reference for %q", referenceName)
	}

	return reference.Hash(), nil
}

func (gr *GitRepo) getCommitHashForTagHash(tagHash plumbing.Hash, repo *git.Repository) (plumbing.Hash, error) {
	tagObject, err := repo.TagObject(tagHash)
	switch err {
	case plumbing.ErrObjectNotFound:
		// The tag is a lightweight tag, which does not need to be resolved further
		return tagHash, nil
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

func (gr *GitRepo) String() string {
	return fmt.Sprintf("%s: %s@%s", gr.Name, gr.Url, gr.Ref)
}
