package command_build

import (
	"fmt"

	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/solidDoWant/distrobuilder/internal/utils"
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
	Action: flags.GitRefValidator,
}

var toolchainDirectoryPathFlag = &cli.PathFlag{
	Name:     "toolchain-directory-path",
	Usage:    "path to the directory containing the toolchain (clang, clang++, etc.) binaries",
	Aliases:  []string{"T"},
	Required: true,
	Action:   flags.ExistingDirValidator,
}

var targetTripletFlag = &cli.StringFlag{
	Name:    "target-triplet",
	Usage:   "triplet that the build should target",
	Aliases: []string{"t"},
	Value:   fmt.Sprintf("%s-pc-linux-musl", utils.GetTripletMachineValue()),
	Action:  flags.TripletValidator,
}

var rootFSDirectoryPathFlag = &cli.PathFlag{
	Name:     "root-fs-directory-path",
	Usage:    "path to the root filesystem directory of the targeted system",
	Aliases:  []string{"R"},
	Required: true,
	Action:   flags.ExistingDirValidator,
}

var configPathFlag = &cli.PathFlag{
	Name:     "config-file-path",
	Usage:    "path to .config Kconfig file",
	Aliases:  []string{"a"},
	Required: true,
	Action:   flags.ExistingFileValidator,
}
