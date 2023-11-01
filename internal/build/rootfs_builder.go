package build

import (
	"fmt"
	"path"

	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
)

type IRootFSBuilder interface {
	SetRootFSDirectoryPath(string)
	GetRootFSDirectoryPath() string
}

type RootFSBuilder struct {
	RootFSDirectoryPath string
}

func (rfsb *RootFSBuilder) SetRootFSDirectoryPath(rootFSDirectoryPath string) {
	rfsb.RootFSDirectoryPath = rootFSDirectoryPath
}

func (rfsb *RootFSBuilder) GetRootFSDirectoryPath() string {
	return rfsb.RootFSDirectoryPath
}

func (rfsb *RootFSBuilder) GetCMakeOptions() *runners.CMakeOptions {
	return &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"CMAKE_SYSROOT": args.StringValue(rfsb.RootFSDirectoryPath),
		},
	}
}

func (rfsb *RootFSBuilder) GetConfigurenOptions() *runners.ConfigureOptions {
	sysrootFlag := fmt.Sprintf("--sysroot=%s", rfsb.RootFSDirectoryPath)
	return &runners.ConfigureOptions{
		AdditionalArgs: map[string]args.IValue{
			"CFLAGS":   args.SeparatorValues(" ", sysrootFlag),
			"CXXFLAGS": args.SeparatorValues(" ", sysrootFlag),
			"LDFLAGS":  args.SeparatorValues(" ", sysrootFlag),
		},
	}
}

func (rfsb *RootFSBuilder) GetGenericRunnerOptions() *runners.GenericRunnerOptions {
	return &runners.GenericRunnerOptions{
		EnvironmentVariables: map[string]args.IValue{
			// Path is set to so that pkg-config can find the libraries
			"PKG_CONFIG_PATH":        args.SeparatorValues(";", path.Join(rfsb.RootFSDirectoryPath, "usr", "lib", "pkgconfig")),
			"PKG_CONFIG_SYSROOT_DIR": args.StringValue(rfsb.RootFSDirectoryPath),
		},
	}
}
