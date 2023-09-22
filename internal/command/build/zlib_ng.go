package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

type ZlibNgCommand struct{}

func (zngc *ZlibNgCommand) GetCommand() *cli.Command {
	return &cli.Command{
		Name: "zlib-ng",
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

func (zngc *ZlibNgCommand) GetBuilder(cliCtx *cli.Context) (build.Builder, error) {
	builder := &build.ZlibNg{}

	return builder, nil
}
