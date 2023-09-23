package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

type MuslLibcCommand struct{}

func (mlc *MuslLibcCommand) GetCommand() *cli.Command {
	return &cli.Command{
		Name: "musl-libc",
		Flags: []cli.Flag{
			sourceDirectoryPathFlag,
			outputDirectoryPathFlag,
			gitRefFlag,
			toolchainDirectoryPathFlag,
			targetTripletFlag,
			rootFSDirectoryPathFlag,
		},
	}
}

func (mlc *MuslLibcCommand) GetBuilder(cliCtx *cli.Context) (build.IBuilder, error) {
	return &build.MuslLibc{}, nil
}
