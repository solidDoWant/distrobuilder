package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewLibtoolCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "libtool",
		Builder: build.NewLibtool(),
	}
}
