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
	"golang.org/x/exp/slices"
)

const compressionLibrary = "zstd"

type IToolchainRequiredBuilder interface {
	ITargetTripletBuilder
	SetToolchainDirectory(string)
	GetToolchainDirectory() string
}

// TODO consider implementing some kind of validation interface for builders
// to ensure that required parameters are set.
// This is especially important if the tool is intended to be useable as a Go
// library at some point.

type ToolchainRequiredBuilder struct {
	TargetTripletBuilder
	ToolchainPath string
}

func (trb *ToolchainRequiredBuilder) SetToolchainDirectory(toolchainDirectory string) {
	trb.ToolchainPath = toolchainDirectory
}

func (trb *ToolchainRequiredBuilder) GetToolchainDirectory() string {
	return trb.ToolchainPath
}

func (trb *ToolchainRequiredBuilder) GetPathForTool(tool string) string {
	return path.Join(trb.GetToolchainBinDirectory(), tool)
}

func (trb *ToolchainRequiredBuilder) GetToolchainBinDirectory() string {
	return path.Join(trb.ToolchainPath, "usr", "bin")
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

func (trb *ToolchainRequiredBuilder) GetConfigurenOptions() *runners.ConfigureOptions {
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

func (trb *ToolchainRequiredBuilder) GetMesonOptions() *runners.MesonOptions {
	compilerFlags := args.SeparatorValues(" ", fmt.Sprintf("-gz=%s", compressionLibrary), "-v")
	linkerPath := args.StringValue(trb.GetPathForTool("ld.lld"))

	return &runners.MesonOptions{
		CrossFile: map[string]map[string]args.IValue{
			"properties": {
				"needs_exe_wrapper": args.TrueValue(),
			},
			"binaries": {
				"c":          args.StringValue(trb.GetPathForTool("clang")),
				"c_args":     compilerFlags,
				"c_ld":       linkerPath,
				"cpp":        args.StringValue(trb.GetPathForTool("clang++")),
				"cpp_args":   compilerFlags,
				"cpp_ld":     linkerPath,
				"strip":      args.StringValue(trb.GetPathForTool("strip")),
				"pkg-config": args.StringValue("pkg-config"), // Use the normal pkg-config binary
			},
			"host_machine": {
				"system":     args.StringValue("linux"),
				"kernel":     args.StringValue("linux"),
				"cpu":        args.StringValue(trb.Triplet.Machine),
				"cpu_family": args.StringValue(trb.Triplet.Machine),
				"endian":     args.StringValue("little"), // TODO don't hardcode this. Not sure how to pull it from clang.
			},
		},
		NativeFile: map[string]map[string]args.IValue{
			"properties": {
				"needs_exe_wrapper": args.FalseValue(),
			},
			"binaries": {
				"c":          args.StringValue("clang"),
				"cpp":        args.StringValue("clang++"),
				"pkg-config": args.StringValue("pkg-config"),
			},
		},
		Options: map[string]args.IValue{
			"strip": args.TrueValue(),
		},
	}
}

func (trb *ToolchainRequiredBuilder) GetGenericRunnerOptions() *runners.GenericRunnerOptions {
	return &runners.GenericRunnerOptions{
		EnvironmentVariables: map[string]args.IValue{
			// Path is set to ensure that builds use toolchain tools when not prefixed properly
			"PATH": args.SeparatorValues(os.PathListSeparator, path.Join(trb.ToolchainPath, "usr", "bin"), os.Getenv("PATH")),
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
