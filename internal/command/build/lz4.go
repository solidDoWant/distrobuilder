package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewLZ4Command() *StandardBuilder {
	return &StandardBuilder{
		Name:    "lz4",
		Builder: build.NewLZ4(),
	}
}
