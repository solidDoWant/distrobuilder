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

func (fob *FilesystemOutputBuilder) GetMesonOptions() *runners.MesonOptions {
	return &runners.MesonOptions{
		Options: map[string]args.IValue{
			"prefix": args.StringValue("/"), // This will be relative to the output directory via DESTDIR environment variable
		},
	}
}

func (fob *FilesystemOutputBuilder) GetGenericRunnerOptions() *runners.GenericRunnerOptions {
	return &runners.GenericRunnerOptions{
		EnvironmentVariables: map[string]args.IValue{
			"DESTDIR": args.StringValue(fob.OutputDirectoryPath), // This is needed by some runners (such as make (usually) meson) and should be safe to set for all
		},
	}
}

func (fob *FilesystemOutputBuilder) getInstallPath(installSubdirectory string) string {
	return path.Join(fob.OutputDirectoryPath, installSubdirectory)
}
