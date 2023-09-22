package build

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type CrossLLVM struct {
	SourceBuilder
	FilesystemOutputBuilder
	LLVMGitRef   string
	MuslGitRef   string
	TargetTriple *utils.Triplet
	Vendor       string

	// Vars for validation checking
	hasBuildCompleted bool
	sourceVersion     string
}

func NewCrossLLVM(targetTriplet *utils.Triplet) *CrossLLVM {
	return &CrossLLVM{
		TargetTriple: targetTriplet,
		Vendor:       "distrobuilder",
	}
}

// Build implements Builder.
func (cb *CrossLLVM) Build(ctx context.Context) error {
	slog.Info("Beginning LLVM clang cross-compiler build")

	muslDirectory, err := cb.buildMuslHeaders(ctx)
	defer utils.Close(muslDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to build musl libc headers")
	}

	err = cb.buildCrossLLVM(ctx, path.Join(muslDirectory.Path, "include"))
	if err != nil {
		return trace.Wrap(err, "failed to build LLVM cross-compiler")
	}

	cb.hasBuildCompleted = true
	slog.Info("Build complete!", "output_directory", cb.OutputDirectoryPath)
	return nil
}

func (cb *CrossLLVM) buildCrossLLVM(ctx context.Context, muslHeaderDirectory string) error {
	slog.Info("Starting LLVM build")
	repo := git_source.NewLLVMGitRepo(cb.SourceDirectoryPath, cb.LLVMGitRef)

	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, cb.OutputDirectoryPath)
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return trace.Wrap(err, "failed to setup for Musl headers build")
	}

	cb.OutputDirectoryPath = outputDirectory.Path

	slog.Info("Running CMake to generate Ninja build file")
	err = cb.runCMake(repo.FullDownloadPath(), buildDirectory.Path, muslHeaderDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to create Ninja build file via CMake")
	}

	slog.Info("Building and installing LLVM clang with Ninja")
	err = cb.runNinja(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to build and install LLVM clang via Ninja")
	}

	slog.Info("Creating output symlinks")
	err = cb.addSymlinks()
	if err != nil {
		return trace.Wrap(err, "failed to create build output symlinks")
	}

	slog.Info("Recording info for validation checking")
	err = cb.recordVersion(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to record LLVM source code version")
	}

	return nil
}

func (cb *CrossLLVM) addSymlinks() error {
	links := map[string]string{
		"bin/ld":  "ld.lld",
		"lib/cpp": "../bin/cpp", // Required for historical reasons, see https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s09.html#ftn.idm236092722896
		"bin/cpp": "clang-cpp",
	}

	err := utils.CreateSymlinks(links, path.Join(cb.OutputDirectoryPath, "usr"))
	if err != nil {
		return trace.Wrap(err, "failed to create all build output symlinks")
	}

	return nil
}

// TODO consider moving this out to a separate build target
func (cb *CrossLLVM) buildMuslHeaders(ctx context.Context) (*utils.Directory, error) {
	slog.Info("Starting Musl libc build (headers only)")
	repo := git_source.NewMuslGitRepo(cb.SourceDirectoryPath, cb.MuslGitRef)
	buildDirectory, outputDirectory, err := setupForBuild(ctx, repo, "")
	defer utils.Close(buildDirectory, &err)
	if err != nil {
		return nil, trace.Wrap(err, "failed to setup for Musl headers build")
	}

	err = cb.runMuslConfigure(repo.FullDownloadPath(), buildDirectory.Path, outputDirectory.Path)
	if err != nil {
		return outputDirectory, trace.Wrap(err, "failed to configure musl libc")
	}

	err = cb.runMuslMake(buildDirectory.Path)
	if err != nil {
		return outputDirectory, trace.Wrap(err, "failed to build musl libc headers")
	}

	return outputDirectory, nil
}

func (cb *CrossLLVM) runMuslMake(buildDirectoryPath string) error {
	_, err := runners.Run(&runners.Make{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectoryPath,
		},
		Path:    ".",
		Targets: []string{"install-headers"},
	})

	if err != nil {
		return trace.Wrap(err, "musl libc make build failed")
	}

	return nil
}

func (cb *CrossLLVM) runMuslConfigure(sourceDirectoryPath, buildDirectoryPath, outputDirectoryPath string) error {
	_, err := runners.Run(&runners.Configure{
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectoryPath,
		},
		Options: []*runners.ConfigureOptions{
			{
				AdditionalArgs: map[string]args.IValue{
					"--prefix": args.StringValue(outputDirectoryPath),
					"--srcdir": args.StringValue(sourceDirectoryPath),
					"CC":       args.StringValue("clang"),
					"CXX":      args.StringValue("clang++"),
					"LIBCC":    args.StringValue(" "), //  Set the value to an empty non-zero length string. This ensures that the headers never reference libgcc.
				},
			},
		},
		ConfigurePath: path.Join(sourceDirectoryPath, "configure"),
		HostTriplet:   cb.TargetTriple, // Libc will run on the target (eventually), so headers should be configured with the target rather than the host
		TargetTriplet: cb.TargetTriple,
	})

	if err != nil {
		return trace.Wrap(err, "failed to configure musl")
	}

	return nil
}

func (cb *CrossLLVM) VerifyBuild(ctx context.Context) error {
	clangPath := path.Join(cb.OutputDirectoryPath, "usr", "bin", "clang")
	clangxxPath := path.Join(cb.OutputDirectoryPath, "usr", "bin", "clang++")
	err := (&runners.VersionChecker{
		CommandRunner: runners.CommandRunner{
			Command:   clangPath,
			Arguments: []string{"--version"},
		},
		VersionRegex: fmt.Sprintf("(?m)clang version %s$", runners.SemverRegex),

		VersionChecker: runners.ExactSemverChecker(cb.sourceVersion),
	}).ValidateOrError()

	if err != nil {
		return trace.Wrap(err, "failed to validate that built clang version matches source code version %q", cb.sourceVersion)
	}

	_, err = runners.Run(runners.CommandRunner{
		Command: clangxxPath,
		Arguments: []string{
			"-v", // Verbose logging to help with errors
			"-o", // Get rid of the compiled output
			"/dev/null",
			"-x", // Compile
			"c",
			"-pipe", // Take the input from stdin
			"-",
		},
		Stdin: "int main() {return 0;}",
	})
	if err != nil {
		return trace.Wrap(err, "failed to compile basic C++ test program")
	}

	return nil
}

func (cb *CrossLLVM) getHostTriplet() (string, error) {
	result, err := runners.Run(&runners.CommandRunner{
		Command:   "clang",
		Arguments: []string{"-dumpmachine"},
	})

	trimmedOutput := strings.TrimSpace(result.Stdout)
	if err != nil || trimmedOutput == "" {
		return "", trace.Wrap(err, "failed to query clang for target triplet")
	}

	return trimmedOutput, nil
}

func (cb *CrossLLVM) runCMake(sourceDirectory, buildDirectory, muslHeaderDirectory string) error {
	hostTriplet, err := cb.getHostTriplet()
	if err != nil {
		return trace.Wrap(err, "failed to get target triplet")
	}

	targetTriplet := cb.TargetTriple.String()
	muslHeaderFlag := args.StringValue(fmt.Sprintf("-isystem%s", muslHeaderDirectory))

	_, err = runners.Run(runners.CMake{
		Generator: "Ninja",
		Options: []*runners.CMakeOptions{
			runners.CommonOptions(),
			cb.FilesystemOutputBuilder.GetCMakeOptions("usr"),
			// General variables to configure CMake
			{
				Defines: map[string]args.IValue{
					"CMAKE_C_COMPILER":            args.StringValue("clang"),
					"CMAKE_C_COMPILER_TARGET":     args.StringValue(hostTriplet), // The produced compiler should run on the host (where this build is running), rather than the target
					"CMAKE_C_COMPILER_LAUNCHER":   args.StringValue("ccache"),
					"CMAKE_CXX_COMPILER":          args.StringValue("clang++"),
					"CMAKE_CXX_COMPILER_TARGET":   args.StringValue(hostTriplet),
					"CMAKE_CXX_COMPILER_LAUNCHER": args.StringValue("ccache"),
				},
			},
			// LLVM project wide config
			{
				Undefines: []string{
					"CLANG_VENDOR_UTI",
				},
				Defines: map[string]args.IValue{
					"LLVM_ENABLE_PROJECTS":           args.SeparatorValues(";", "clang", "lld"),                                             // Other projects are not needed for cross compiling
					"LLVM_ENABLE_RUNTIMES":           args.SeparatorValues(";", "compiler-rt", "libcxx", "libcxxabi", "libunwind"),          // Other runtimes are not needed - only C++ is required
					"LLVM_ENABLE_PIC":                args.OnValue(),                                                                        // Required by Musl libc, creates "position independent code"
					"LLVM_ENABLE_LLD":                args.OnValue(),                                                                        // Use the LLVM version of ld
					"LLVM_ENABLE_ZSTD":               args.ForcedOnValue(),                                                                  // This is only supported by newer tools, but can compress debug sections to a much smaller size than zlib while not increasing decompression time
					"LLVM_INSTALL_BINUTILS_SYMLINKS": args.OnValue(),                                                                        // Increase the change of using LLVM tools even if something is misconfigured somewhere
					"LLVM_INSTALL_CCTOOLS_SYMLINKS":  args.OnValue(),                                                                        // Increase the change of using clang even if something is misconfigured somewhere
					"LLVM_HOST_TRIPLE":               args.StringValue(hostTriplet),                                                         //
					"LLVM_TARGET_TRIPLE":             args.StringValue(targetTriplet),                                                       // The _produced_ cross compiler should produce builds for the target
					"LLVM_DEFAULT_TARGET_TRIPLE":     args.StringValue(targetTriplet),                                                       // The _produced_ cross compiler should by default produce builds for the target
					"LLVM_TARGETS_TO_BUILD":          args.SeparatorValues(";", "X86"),                                                      // TODO calculate this based upon host triplet
					"LLVM_APPEND_VC_REV":             args.OnValue(),                                                                        // Include the release version, obtained from Git, in the version output
					"LLVM_CCACHE_BUILD":              args.OnValue(),                                                                        // Useful for development to reduce build times TODO figure out why this isn't working
					"LLVM_PARALLEL_LINK_JOBS":        args.StringValue(fmt.Sprintf("%d", runners.GetCmakeMaxRecommendedParallelLinkJobs())), // Setting this too high will cause the build processes to get OOM killed
					"PACKAGE_VENDOR":                 args.StringValue(cb.Vendor),                                                           // Branding
				},
			},
			// Compiler-rt config
			{
				Defines: map[string]args.IValue{
					// Disable features that are not needed for the cross compiler
					"COMPILER_RT_USE_BUILTINS_LIBRARY": args.OnValue(), // If not set (or off) then compiler-rt will use libgcc
					"COMPILER_RT_USE_LLVM_UNWINDER":    args.OnValue(),
					"COMPILER_RT_CXX_LIBRARY":          args.StringValue("libcxx"),
					"COMPILER_RT_DEFAULT_TARGET_ONLY":  args.OnValue(),
					"COMPILER_RT_INCLUDE_TESTS":        args.OnValue(),
					"COMPILER_RT_BUILD_BUILTINS":       args.OnValue(),
					"COMPILER_RT_BUILD_SANITIZERS":     args.OffValue(),
					"COMPILER_RT_BUILD_MEMPROF":        args.OffValue(),
					"COMPILER_RT_BUILD_LIBFUZZER":      args.OffValue(), // Enabling this will cause the build to fail when LIBCXX_HAS_MUSL_LIBC is enabled
					"COMPILER_RT_BUILD_XRAY":           args.OffValue(), // Enabling this will cause the build to fail when LIBCXX_HAS_MUSL_LIBC is enabled
					"COMPILER_RT_BUILD_ORC":            args.OffValue(), // Enabling this will cause the build to fail when LIBCXX_HAS_MUSL_LIBC is enabled
					// "COMPILER_RT_BUILD_PROFILE": args.OffValue(),	 // Unsure if this is needed or not for optimizing the next compiler stage
				},
			},
			// libunwind options
			{
				Defines: map[string]args.IValue{
					"LIBUNWIND_USE_COMPILER_RT": args.OnValue(),
				},
			},
			// libc++abi config
			{
				Defines: map[string]args.IValue{
					"LIBCXXABI_USE_COMPILER_RT":          args.OnValue(),
					"LIBCXXABI_USE_LLVM_UNWINDER":        args.OnValue(),
					"LIBCXXABI_ADDITIONAL_COMPILE_FLAGS": muslHeaderFlag, // Configure libc++ abi to search the Musl libc include directory
				},
			},
			// libc++ config
			{
				Defines: map[string]args.IValue{
					"LIBCXX_USE_COMPILER_RT":          args.OnValue(),
					"LIBCXX_HAS_MUSL_LIBC":            args.OnValue(), // Required to be able to compile objects referencing Musl libc
					"LIBCXX_HAS_PTHREAD_API":          args.OnValue(),
					"LIBCXX_CXX_ABI":                  args.StringValue("libcxxabi"), // Tell the runtimes to link against LLVM libs rather than GCC
					"LIBCXX_ADDITIONAL_COMPILE_FLAGS": muslHeaderFlag,                // Configure libc++ to search the Musl libc include directory
				},
			},
			// clang config
			{
				Defines: map[string]args.IValue{
					"CLANG_DEFAULT_RTLIB":          args.StringValue("compiler-rt"), // Use the newly built runtime
					"CLANG_DEFAULT_UNWINDLIB":      args.StringValue("libunwind"),
					"CLANG_DEFAULT_CXX_STDLIB":     args.StringValue("libc++"),
					"CLANG_ENABLE_STATIC_ANALYZER": args.OffValue(), // Used for development, not needed for cross-compiling
					"CLANG_ENABLE_ARCMT":           args.OffValue(),
				},
			},
		},
		Path: path.Join(sourceDirectory, "llvm"),
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectory,
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to create generator file for LLVM clang cross-compiler")
	}

	return nil
}

func (cb *CrossLLVM) runNinja(buildDirectory string) error {
	_, err := runners.Run(runners.CommandRunner{
		Command:   "/workspaces/distrobuilder/test.sh",
		Arguments: []string{"ninja", "install"},
		GenericRunner: runners.GenericRunner{
			WorkingDirectory: buildDirectory,
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to build LLVM clang")
	}

	return nil
}

func (cb *CrossLLVM) recordVersion(buildDirectoryPath string) error {
	cacheVars, err := runners.GetCmakeCacheVars(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to get CMake cache vars from build directory %q", buildDirectoryPath)
	}

	majorVersion, ok := cacheVars["CMAKE_PROJECT_VERSION_MAJOR"]
	if !ok {
		return trace.Errorf("failed to read major version from CMake cache values")
	}
	minorVersion, ok := cacheVars["CMAKE_PROJECT_VERSION_MINOR"]
	if !ok {
		return trace.Errorf("failed to read minor version from CMake cache values")
	}
	patchVersion, ok := cacheVars["CMAKE_PROJECT_VERSION_PATCH"]
	if !ok {
		return trace.Errorf("failed to read patch version from CMake cache values")
	}

	cb.sourceVersion = fmt.Sprintf("%s.%s.%s", majorVersion, minorVersion, patchVersion)
	return nil
}

// CheckHostRequirements implements Builder.
func (CrossLLVM) CheckHostRequirements() error {
	// Pulled from https://llvm.org/docs/GettingStarted.html#software
	requiredCommands := []string{
		"cmake",
		"python",
		"ninja", // Note that the linked page specifies GNU Make but Ninja is used by this tool instead
		"ar",
		"bzip2",
		"bunzip2",
		"chmod",
		"cat",
		"cp",
		"date",
		"echo",
		"egrep",
		"find",
		"grep",
		"gzip",
		"gunzip",
		"install",
		"mkdir",
		"mv",
		"ranlib",
		"rm",
		"sed",
		"sh",
		"tar",
		"test",
		"unzip",
		"zip",
		"clang",
		"clang++",
	}

	err := runners.CheckRequiredCommandsExist(requiredCommands)
	if err != nil {
		return trace.Wrap(err, "failed to verify that all required commands exist")
	}

	versionCheckers := []runners.VersionChecker{
		{
			CommandRunner: runners.CommandRunner{
				Command:   "cmake",
				Arguments: []string{"--version"},
			},
			VersionRegex:   fmt.Sprintf("(?m)^cmake version %s$", runners.SemverRegex),
			VersionChecker: runners.MinSemverChecker("3.20.0"),
		},
		{
			CommandRunner: runners.CommandRunner{
				Command:   "python",
				Arguments: []string{"--version"},
			},
			VersionRegex:   fmt.Sprintf("(?m)^Python %s$", runners.SemverRegex),
			VersionChecker: runners.MinSemverChecker("3.6.0"),
		},
	}

	for _, versionChecker := range versionCheckers {
		err := versionChecker.ValidateOrError()
		if err != nil {
			return trace.Wrap(err, "failed to validate host requirement version for %q", versionChecker.PrettyPrint())
		}
	}

	return nil
}
