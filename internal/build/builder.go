package build

import (
	"context"
)

type Builder interface {
	CheckHostRequirements() error
	Build(context.Context) (string, error)
	VerifyBuild(context.Context, string) error
	// RequiredSpace() int	// TODO
}
