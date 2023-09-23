package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

type XZCommand struct {
	StandardBuilder
}

func NewXZCommand() *ZstdCommand {
	return &ZstdCommand{
		StandardBuilder: StandardBuilder{
			Name:    "xz",
			Builder: build.NewXZ(),
		},
	}
}
