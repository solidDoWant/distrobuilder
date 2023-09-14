package git_source

import (
	"os"
	"path"

	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLLVMRef string = "refs/tags/llvmorg-16.0.6"
	LLVMRepoUrl    string = "https://github.com/llvm/llvm-project.git"
)

type LLVMGitRepo struct {
	*source.GitRepo
}

func NewLLVMGitRepo(repoDirectoryPath, ref string) *LLVMGitRepo {
	if ref == "" {
		ref = DefaultLLVMRef
	}

	if repoDirectoryPath == "" {
		repoDirectoryPath = path.Join(os.TempDir(), "source", "llvm")
	}

	return &LLVMGitRepo{
		GitRepo: source.NewGitRepo(repoDirectoryPath, LLVMRepoUrl, ref),
	}
}
