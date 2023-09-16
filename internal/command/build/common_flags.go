package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/urfave/cli/v2"
)

var sourceDirectoryPathFlag = &cli.PathFlag{
	Name:    "source-directory-path",
	Usage:   "directory path that should be used for storing source files",
	Aliases: []string{"S"},
	Value:   "",
}

var outputDirectoryPathFlag = &cli.PathFlag{
	Name:    "output-directory-path",
	Usage:   "path where the build outputs should be placed",
	Aliases: []string{"O"},
	Value:   "",
}

var gitRefFlag = &cli.StringFlag{
	Name:   "git-ref",
	Usage:  "the fully qualified Git ref to build the Musl from",
	Value:  "refs/tags/v1.2.4",
	Action: flags.GitRefValidator,
}

var toolchainDirectoryPathFlag = &cli.PathFlag{
	Name:     "toolchain-directory-path",
	Usage:    "path to the directory containing the toolchain (clang, clang++, etc.) binaries",
	Aliases:  []string{"T"},
	Required: true,
	Action:   flags.ExistingDirValidator,
}
