package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

type StandardBuilder struct {
	Name    string
	Builder build.IBuilder
}

func (sb *StandardBuilder) GetCommand() *cli.Command {
	return &cli.Command{
		Name: sb.Name,
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

func (sb *StandardBuilder) GetBuilder(cliCtx *cli.Context) (build.IBuilder, error) {
	return sb.Builder, nil
}
