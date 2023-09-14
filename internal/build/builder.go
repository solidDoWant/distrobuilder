package build

import (
	"context"
)

type Builder interface {
	CheckHostRequirements() error
	Build(context.Context) error
	VerifyBuild(context.Context, string) error
	// RequiredSpace() int	// TODO
}

type SourceBuilder struct {
	SourceDirectoryPath string
}

type FilesystemOutputBuilder struct {
	OutputDirectoryPath string
}
