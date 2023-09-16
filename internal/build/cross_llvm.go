package build

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
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

	slog.Info("Recording info for validation checking")
	err = cb.recordVersion(buildDirectory.Path)
	if err != nil {
		return trace.Wrap(err, "failed to record LLVM source code version")
	}

	return nil
}

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
		CCompiler:           "clang",
		CppCompiler:         "clang++",
		InstallPath:         outputDirectoryPath,
		SourceDirectoryPath: sourceDirectoryPath,
		ConfigurePath:       path.Join(sourceDirectoryPath, "configure"),
		HostTriplet:         cb.TargetTriple, // Libc will run on the target (eventually), so headers should be configured with the target rather than the host
		TargetTriplet:       cb.TargetTriple,
	})

	if err != nil {
		return trace.Wrap(err, "failed to configure musl")
	}

	return nil
}

func (cb *CrossLLVM) VerifyBuild(ctx context.Context) error {
	err := (&runners.VersionChecker{
		CommandRunner: runners.CommandRunner{
			Command:   path.Join(cb.OutputDirectoryPath, "bin", "clang"),
			Arguments: []string{"--version"},
		},
		VersionRegex: fmt.Sprintf("(?m)clang version %s$", runners.SemverRegex),

		VersionChecker: runners.ExactSemverChecker(cb.sourceVersion),
	}).ValidateOrError()

	if err != nil {
		return trace.Wrap(err, "failed to validate that built clang version matches source code version %q", cb.sourceVersion)
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

	targetTriplet, err := cb.TargetTriple.AsString()
	if err != nil {
		return trace.Wrap(err, "failed to convert target triplet %v to string", cb.TargetTriple)
	}

	muslHeaderFlag := fmt.Sprintf("-isystem%s", muslHeaderDirectory)

	_, err = runners.Run(runners.CMake{
		Generator: "Ninja",
		Defines: map[string]string{
			"LLVM_HOST_TRIPLE":                   hostTriplet,
			"LLVM_TARGET_TRIPLE":                 targetTriplet, // The _produced_ cross compiler should produce builds for the target
			"LLVM_DEFAULT_TARGET_TRIPLE":         targetTriplet, // The _produced_ cross compiler should by default produce builds for the target
			"CMAKE_SYSTEM_NAME":                  "Linux",       // This will enable cross compiling when explicitly set
			"CMAKE_C_COMPILER_TARGET":            hostTriplet,   // The produced compiler should run on the host (where this build is running), rather than the target
			"CMAKE_CXX_COMPILER_TARGET":          hostTriplet,
			"CMAKE_C_COMPILER":                   "clang",
			"CMAKE_CXX_COMPILER":                 "clang++",
			"CMAKE_BUILD_TYPE":                   "Release",
			"LLVM_ENABLE_PROJECTS":               "clang;clang-tools-extra;lld",
			"LLVM_ENABLE_RUNTIMES":               "compiler-rt;libcxx;libcxxabi;libunwind",
			"CMAKE_INSTALL_PREFIX":               cb.OutputDirectoryPath,
			"LLVM_TARGETS_TO_BUILD":              "X86",
			"LLVM_APPEND_VC_REV":                 "ON",
			"LLVM_ENABLE_PIC":                    "ON",       // Required by musl
			"LLVM_ENABLE_LLD":                    "ON",       // Use the LLVM version of ld
			"LLVM_ENABLE_ZSTD":                   "FORCE_ON", // This is only supported by newer tools, but generally has much better performance than zlib
			"LLVM_INSTALL_BINUTILS_SYMLINKS":     "ON",       // Increase the change of using LLVM tools even if something is misconfigured somewhere
			"LLVM_INSTALL_CCTOOLS_SYMLINKS":      "ON",       // Increase the change of using clang even if something is misconfigured somewhere
			"LLVM_INSTALL_UTILS":                 "ON",
			"LLVM_PARALLEL_LINK_JOBS":            fmt.Sprintf("%d", runners.GetCmakeMaxRecommendedParallelLinkJobs()), // Setting this too high will cause the build processes to get OOM killed
			"LLVM_CCACHE_BUILD":                  "ON",                                                                // Useful for development to reduce build times TODO figure out why this isn't working
			"CMAKE_C_COMPILER_LAUNCHER":          "ccache",
			"CMAKE_CXX_COMPILER_LAUNCHER":        "ccache",
			"COMPILER_RT_BUILD_SANITIZERS":       "OFF", // Disable features that are not needed for the cross compiler
			"COMPILER_RT_BUILD_MEMPROF":          "OFF",
			"COMPILER_RT_BUILD_LIBFUZZER":        "OFF", // Enabling this will cause the build to fail when LIBCXX_HAS_MUSL_LIBC is enabled
			"COMPILER_RT_BUILD_XRAY":             "OFF", // Enabling this will cause the build to fail when LIBCXX_HAS_MUSL_LIBC is enabled
			"COMPILER_RT_BUILD_ORC":              "OFF",
			"COMPILER_RT_BUILD_PROFILE":          "OFF",
			"CLANG_DEFAULT_RTLIB":                "compiler-rt", // Use the newly built runtimes
			"CLANG_DEFAULT_UNWINDLIB":            "libunwind",
			"CLANG_DEFAULT_CXX_STDLIB":           "libc++",
			"LIBCXX_HAS_MUSL_LIBC":               "ON",
			"LIBCXX_CXX_ABI":                     "libcxxabi", // Tell the runtimes to link against LLVM libs rather than GCC
			"LIBCXX_USE_COMPILER_RT":             "ON",
			"LIBCXX_ADDITIONAL_COMPILE_FLAGS":    muslHeaderFlag, // Configure libc++ to search the Musl libc include directory
			"LIBCXXABI_ADDITIONAL_COMPILE_FLAGS": muslHeaderFlag, // Configure libc++ abi to search the Musl libc include directory
			"LIBCXXABI_USE_LLVM_UNWINDER":        "ON",
			"LIBCXXABI_USE_COMPILER_RT":          "ON",
			"LIBUNWIND_USER_COMPILER_RT":         "ON",
			"CLANG_VENDOR":                       cb.Vendor, // Branding
			"LLD_VENDOR":                         cb.Vendor,
		},
		Undefines: []string{
			"CLANG_VENDOR_UTI",
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
