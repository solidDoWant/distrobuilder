package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

type LinuxHeadersCommand struct{}

func (lh *LinuxHeadersCommand) GetCommand() *cli.Command {
	return &cli.Command{
		Name: "linux-headers",
		Flags: []cli.Flag{
			gitRefFlag,
			outputDirectoryPathFlag,
			sourceDirectoryPathFlag,
			targetTripletFlag,
		},
	}
}

func (lh *LinuxHeadersCommand) GetBuilder(cliCtx *cli.Context) (build.IBuilder, error) {
	return &build.LinuxHeaders{}, nil
}
