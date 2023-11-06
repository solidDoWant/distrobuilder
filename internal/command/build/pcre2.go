package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewPCRE2Command() *StandardBuilder {
	return &StandardBuilder{
		Name:    "pcre2",
		Builder: build.NewPCRE2(),
	}
}
