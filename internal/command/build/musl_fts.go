package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewMuslFTSCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "musl-fts",
		Builder: build.NewMuslFTS(),
	}
}
