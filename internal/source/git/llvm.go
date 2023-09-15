package git_source

import (
	"github.com/solidDoWant/distrobuilder/internal/source"
)

const (
	DefaultLLVMRef string = "refs/tags/llvmorg-16.0.6"
	LLVMRepoUrl    string = "https://github.com/llvm/llvm-project.git"
)

func NewLLVMGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	if ref == "" {
		ref = DefaultLLVMRef
	}

	return source.NewGitRepo("llvm", LLVMRepoUrl, ref)
}
