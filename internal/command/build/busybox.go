package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

type BusyBoxBuilder struct {
	*StandardBuilder
}

func NewBusyBoxCommand() *BusyBoxBuilder {
	return &BusyBoxBuilder{
		&StandardBuilder{
			Name:    "busybox",
			Builder: build.NewBusyBox(),
		},
	}
}

func (bb *BusyBoxBuilder) GetCommand() *cli.Command {
	standardCommand := bb.StandardBuilder.GetCommand()
	standardCommand.Flags = append(standardCommand.Flags, configPathFlag)
	return standardCommand
}
