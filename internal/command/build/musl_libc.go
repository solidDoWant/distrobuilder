package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/urfave/cli/v2"
)

type MuslLibcCommand struct{}

var linuxHeaderDirectoryPathFlag = &cli.PathFlag{
	Name:     "linux-header-directory-path",
	Usage:    "Path to the directory containing the Linux kernel headers",
	Required: true,
	Action:   flags.ExistingDirValidator,
}

func (mlc *MuslLibcCommand) GetCommand() *cli.Command {
	return &cli.Command{
		Name: "musl-libc",
		Flags: []cli.Flag{
			sourceDirectoryPathFlag,
			outputDirectoryPathFlag,
			gitRefFlag,
			toolchainDirectoryPathFlag,
			linuxHeaderDirectoryPathFlag,
		},
	}
}

func (mlc *MuslLibcCommand) GetBuilder(cliCtx *cli.Context) (build.Builder, error) {
	builder := &build.MuslLibc{}
	builder.KernelHeaderDirectoryPath = cliCtx.Path(linuxHeaderDirectoryPathFlag.Name)

	return builder, nil
}
