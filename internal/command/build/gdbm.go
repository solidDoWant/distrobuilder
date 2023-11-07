package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewGDBMCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "gdbm",
		Builder: build.NewGDBM(),
	}
}
