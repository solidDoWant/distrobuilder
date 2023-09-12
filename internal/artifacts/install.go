package artifacts

import "context"

type InstallOptions struct {
	InstallPath string
	SourcePath  string
}

type Install interface {
	Install(context.Context, *InstallOptions) error
}
