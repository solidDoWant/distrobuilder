package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewXZCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "xz",
		Builder: build.NewXZ(),
	}
}
