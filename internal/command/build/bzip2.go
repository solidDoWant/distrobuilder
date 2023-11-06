package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewBzip2Command() *StandardBuilder {
	return &StandardBuilder{
		Name:    "bzip2",
		Builder: build.NewBzip2(),
	}
}
