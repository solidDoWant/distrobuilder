package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewMuslLibcCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "musl-libc",
		Builder: build.NewMuslLibc(),
	}
}
