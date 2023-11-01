package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

type LinuxKernelBuilder struct {
	*StandardBuilder
}

func NewLinuxKernelCommand() *LinuxKernelBuilder {
	return &LinuxKernelBuilder{
		StandardBuilder: &StandardBuilder{
			Name:    "linux-kernel",
			Builder: build.NewLinuxKernel(),
		},
	}
}

func (lk *LinuxKernelBuilder) GetCommand() *cli.Command {
	standardCommand := lk.StandardBuilder.GetCommand()
	standardCommand.Flags = append(standardCommand.Flags, configPathFlag)
	return standardCommand
}
