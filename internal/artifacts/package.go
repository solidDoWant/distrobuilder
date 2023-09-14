package artifacts

import "context"

type Package interface {
	Package(context.Context) (string, error)
}

type FilesystemOutputPackager interface {
	SetOutputFilePath(string)
}
