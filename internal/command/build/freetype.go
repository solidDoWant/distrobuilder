package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewFreeTypeCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "freetype",
		Builder: build.NewFreeType(),
	}
}
