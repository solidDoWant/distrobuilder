package build

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
	"golang.org/x/exp/slices"
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
			"LIBCC":    args.StringValue("-lclang_rt.builtins"), // Replaces libgcc.a
		},
	}
}

func (trb *ToolchainRequiredBuilder) GetEnvironmentVariables() map[string]string {
	return map[string]string{
		// Path is set to ensure that builds use toolchain tools when not prefixed properly
		"PATH": fmt.Sprintf("%s%c%s", path.Join(trb.ToolchainPath, "bin"), os.PathListSeparator, os.Getenv("PATH")),
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

func (trb *ToolchainRequiredBuilder) VerifyTargetElfFile(targetExecutablePath string) error {
	file, err := elf.Open(targetExecutablePath)
	if err != nil {
		return trace.Wrap(err, "failed to open executable for validation")
	}

	executableMachine := strings.ToLower(strings.TrimPrefix(file.Machine.String(), "EM_"))
	targetMachine := strings.ToLower(trb.Triplet.Machine)
	if executableMachine != targetMachine {
		return trace.Errorf("the executable machine type %q does not match desired target machine type %q", executableMachine, targetMachine)
	}

	// TODO check if binary is position independent

	interpreterSection := pie.Of(file.Progs).Filter(func(programSection *elf.Prog) bool { return programSection.Type == elf.PT_INTERP }).First()
	if interpreterSection == nil {
		// Assume that the binary is statically linked
		// TODO verify that this is a valid test
		return nil
	}

	buffer := make([]byte, interpreterSection.Filesz-1) // The last character is a null termination character, don't read it
	_, err = interpreterSection.ReadAt(buffer, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return trace.Wrap(err, "failed to read entire interpreter section from the executable")
	}

	interpreterPath := string(buffer)
	desiredInterpreterPaths := []string{path.Join("/lib", trb.Triplet.GetDynamicLoaderName()), path.Join("/usr", "/lib", trb.Triplet.GetDynamicLoaderName())}
	if !slices.Contains(desiredInterpreterPaths, interpreterPath) {
		return trace.Errorf("the executable interpreter %q does not match one of expected value %v", interpreterPath, desiredInterpreterPaths)
	}

	// TODO check .dynamic section for possible host libs

	return nil
}
