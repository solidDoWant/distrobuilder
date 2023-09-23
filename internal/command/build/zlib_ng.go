package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

type ZlibNgCommand struct {
	StandardBuilder
}

func NewZlibNgCommand() *ZlibNgCommand {
	return &ZlibNgCommand{
		StandardBuilder: StandardBuilder{
			Name:    "zlib-ng",
			Builder: build.NewZLibNg(),
		},
	}
}
