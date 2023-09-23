package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewZlibNgCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "zlib-ng",
		Builder: build.NewZLibNg(),
	}
}
