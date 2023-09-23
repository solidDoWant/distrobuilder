package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

type RootFilesystemCommand struct{}

func (rfc *RootFilesystemCommand) GetCommand() *cli.Command {
	return &cli.Command{
		Name: "root-filesystem",
		Flags: []cli.Flag{
			outputDirectoryPathFlag,
		},
	}
}

func (rfc *RootFilesystemCommand) GetBuilder(cliCtx *cli.Context) (build.IBuilder, error) {
	return &build.RootFilesystem{}, nil
}
