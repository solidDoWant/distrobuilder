package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewLibFUSECommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "libfuse",
		Builder: build.NewLibFUSE(),
	}
}
