package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewLibiconvCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "libiconv",
		Builder: build.NewLibiconv(),
	}
}
