package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewZstdCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "zstd",
		Builder: build.NewZstd(),
	}
}
