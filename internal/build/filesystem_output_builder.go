package build

import (
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
)

type IFilesystemOutputBuilder interface {
	SetOutputDirectoryPath(string)
	GetOutputDirectoryPath() string
}

type FilesystemOutputBuilder struct {
	OutputDirectoryPath string
}

func (fob *FilesystemOutputBuilder) SetOutputDirectoryPath(outputDirectoryPath string) {
	fob.OutputDirectoryPath = outputDirectoryPath
}

func (fob *FilesystemOutputBuilder) GetOutputDirectoryPath() string {
	return fob.OutputDirectoryPath
}

func (fob *FilesystemOutputBuilder) GetCMakeOptions(installSubdirectory string) *runners.CMakeOptions {
	return &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"CMAKE_INSTALL_PREFIX": args.StringValue(fob.getInstallPath(installSubdirectory)),
		},
	}
}

func (fob *FilesystemOutputBuilder) GetConfigurenOptions(installSubdirectory string) *runners.ConfigureOptions {
	return &runners.ConfigureOptions{
		AdditionalArgs: map[string]args.IValue{
			"--prefix": args.StringValue(fob.getInstallPath(installSubdirectory)),
		},
	}
}

func (fob *FilesystemOutputBuilder) getInstallPath(installSubdirectory string) string {
	return path.Join(fob.OutputDirectoryPath, installSubdirectory)
}
