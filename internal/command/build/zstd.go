package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

type ZstdCommand struct {
	StandardBuilder
}

func NewZstdCommand() *ZstdCommand {
	return &ZstdCommand{
		StandardBuilder: StandardBuilder{
			Name:    "zstd",
			Builder: build.NewZstd(),
		},
	}
}
