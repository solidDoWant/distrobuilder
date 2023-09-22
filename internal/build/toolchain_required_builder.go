package build

import (
	"fmt"
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

const compressionLibrary = "zstd"

type IToolchainRequiredBuilder interface {
	SetToolchainDirectory(string)
	GetToolchainDirectory() string
	SetTargetTriplet(*utils.Triplet)
	GetTargetTriplet() *utils.Triplet
}

// TODO consider implementing some kind of validation interface for builders
// to ensure that required parameters are set.
// This is especially important if the tool is intended to be useable as a Go
// library at some point.

type ToolchainRequiredBuilder struct {
	ToolchainPath string
	Triplet       *utils.Triplet
}

func (trb *ToolchainRequiredBuilder) SetToolchainDirectory(toolchainDirectory string) {
	trb.ToolchainPath = toolchainDirectory
}

func (trb *ToolchainRequiredBuilder) SetTargetTriplet(triplet *utils.Triplet) {
	trb.Triplet = triplet
}

func (trb *ToolchainRequiredBuilder) GetTargetTriplet() *utils.Triplet {
	return trb.Triplet
}

func (trb *ToolchainRequiredBuilder) GetToolchainDirectory() string {
	return trb.ToolchainPath
}

func (trb *ToolchainRequiredBuilder) GetPathForTool(tool string) string {
	return path.Join(trb.ToolchainPath, "usr", "bin", tool)
}

func (trb *ToolchainRequiredBuilder) GetCMakeOptions() *runners.CMakeOptions {
	compilerFlags := args.SeparatorValues(" ", fmt.Sprintf("-gz=%s", compressionLibrary)) // "-gz" tells the compiler to compress debug sections using the specified library

	return &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"CMAKE_C_COMPILER":   args.StringValue(trb.GetPathForTool("clang")),
			"CMAKE_CXX_COMPILER": args.StringValue(trb.GetPathForTool("clang++")),
			"CMAKE_LINKER":       args.StringValue(trb.GetPathForTool("ld.lld")), // This should be detected automatically but setting it manually ensures that it will be right
			"CMAKE_C_FLAGS":      compilerFlags,
			"CMAKE_CXX_FLAGS":    compilerFlags,
		},
	}
}

func (trb *ToolchainRequiredBuilder) GetConfigurenOptions(installSubdirectory string) *runners.ConfigureOptions {
	compilerFlags := args.SeparatorValues(" ", fmt.Sprintf("-gz=%s", compressionLibrary), fmt.Sprintf("-fuse-ld=%s", trb.GetPathForTool("ld.lld")))

	return &runners.ConfigureOptions{
		AdditionalArgs: map[string]args.IValue{
			"CC":       args.StringValue(trb.GetPathForTool("clang")),
			"CXX":      args.StringValue(trb.GetPathForTool("clang++")),
			"CFLAGS":   compilerFlags,
			"CXXFLAGS": compilerFlags,
		},
	}
}

func (trb *ToolchainRequiredBuilder) CheckToolsExist() error {
	requiredCommands := []string{
		"clang",
		"clang++",
		"ld.lld",
	}

	for i := range requiredCommands {
		requiredCommands[i] = trb.GetPathForTool(requiredCommands[i])
	}

	err := runners.CheckRequiredCommandsExist(requiredCommands)
	if err != nil {
		return trace.Wrap(err, "failed to verify that all required commands exist")
	}

	return nil
}
