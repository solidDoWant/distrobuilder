package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
)

func NewLibreSSLCommand() *StandardBuilder {
	return &StandardBuilder{
		Name:    "libressl",
		Builder: build.NewLibreSSL(),
	}
}
