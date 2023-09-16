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
			sourceDirectoryPathFlag,
			outputDirectoryPathFlag,
		},
	}
}

func (lh *LinuxHeadersCommand) GetBuilder(cliCtx *cli.Context) (build.Builder, error) {
	return &build.LinuxHeaders{}, nil
}
