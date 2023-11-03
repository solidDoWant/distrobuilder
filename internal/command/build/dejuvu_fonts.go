package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewDejaVuFontsCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "dejavu-fonts",
		Builder: build.NewDejaVuFonts(),
	}
}
