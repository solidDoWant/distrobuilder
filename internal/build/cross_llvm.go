package build

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/source"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type CrossLLVM struct {
	SourceBuilder
	FilesystemOutputBuilder
	GitRef       string
	TargetTriple *utils.Triplet
	Vendor       string

	// Vars for validation checking
	hasBuildCompleted bool
	// buildOutputPath   string
	builtVersion string
}

func NewCrossLLVM(gitRef string, targetTriplet *utils.Triplet) *CrossLLVM {
	if gitRef == "" {
		gitRef = "HEAD"
	}

	return &CrossLLVM{
		GitRef:       gitRef,
		TargetTriple: targetTriplet,
		Vendor:       "distrobuilder",
	}
}

// Build implements Builder.
func (cb *CrossLLVM) Build(ctx context.Context) error {
	slog.Info("Beginning LLVM clang cross-compiler build")

	source := source.NewGitRepo(cb.SourceDirectoryPath, "https://github.com/llvm/llvm-project", cb.GitRef)
	sourceDirectory := source.FullDownloadPath()
	slog.Info("Cloning git repo", "repo", source.PrettyName(), "download_path", sourceDirectory)
	err := source.Download(ctx)
	if err != nil {
		return trace.Wrap(err, "failed to clone LLVM from %q", source.PrettyName())
	}
	defer func() {
		cleanupErr := source.Cleanup()
		if cleanupErr != nil && err == nil {
			err = trace.Wrap(err, "failed to cleanup git source")
		}
	}()

	slog.Info("Creating build and output directories")
	buildDirectory, err := utils.EnsureTempDirectoryExists()
	if err != nil {
		return trace.Wrap(err, "failed to create build directory at %q", buildDirectory)
	}
	defer func() {
		cleanupErr := os.RemoveAll(buildDirectory)
		if cleanupErr != nil && err == nil {
			err = trace.Wrap(err, "failed to remove build directory")
		}
	}()

	if cb.OutputDirectoryPath == "" {
		cb.OutputDirectoryPath = utils.GetTempDirectoryPath()
	}
	_, err = utils.EnsureDirectoryExists(cb.OutputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to create output directory at %q", cb.OutputDirectoryPath)
	}
	slog.Debug("Created build and output directories", "build_directory", buildDirectory, "output_directory", cb.OutputDirectoryPath)

	slog.Info("Running CMake to generate Ninja build file")
	err = cb.runCMake(sourceDirectory, buildDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to create Ninja build file via CMake")
	}

	slog.Info("Building and installing LLVM clang with Ninja")
	err = cb.runNinja(buildDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to build and install LLVM clang via Ninja")
	}

	slog.Info("Recording info for validation checking")
	err = cb.recordVersion(buildDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to record LLVM source code version")
	}

	cb.hasBuildCompleted = true

	slog.Info("Build complete!", "output_directory", cb.OutputDirectoryPath)
	return nil
}

func (cb *CrossLLVM) VerifyBuild(ctx context.Context) error {
	err := (&runners.VersionChecker{
		CommandRunner: runners.CommandRunner{
			Command:   path.Join(cb.OutputDirectoryPath, "bin", "clang"),
			Arguments: []string{"--version"},
		},
		VersionRegex: fmt.Sprintf("(?m)clang version %s$", runners.SemverRegex),

		VersionChecker: runners.ExactSemverChecker(cb.builtVersion),
	}).ValidateOrError()

	if err != nil {
		return trace.Wrap(err, "failed to validate that built clang version matches source code version %q", cb.builtVersion)
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

func (cb *CrossLLVM) runCMake(sourceDirectory, buildDirectory string) error {
	hostTriplet, err := cb.getHostTriplet()
	if err != nil {
		return trace.Wrap(err, "failed to get target triplet")
	}

	targetTriplet, err := cb.TargetTriple.AsString()
	if err != nil {
		return trace.Wrap(err, "failed to convert target triplet %v to string", cb.TargetTriple)
	}

	commonFlags := strings.Join(
		[]string{
			fmt.Sprintf("-DLLVM_REVISION=%q", cb.GitRef),
			"-DTSAN_VECTORIZE=0", // This disables a compiler-rt feature that does not appear to be supported on x86_64, and will fail builds if enabled
		},
		" ",
	)

	_, err = runners.Run(runners.CMake{
		Generator: "Ninja",
		Defines: []runners.CMakeDefine{
			{Name: "LLVM_HOST_TRIPLE", Value: hostTriplet},
			{Name: "LLVM_TARGET_TRIPLE", Value: targetTriplet},         // The _produced_ cross compiler should produce builds for the target
			{Name: "LLVM_DEFAULT_TARGET_TRIPLE", Value: targetTriplet}, // The _produced_ cross compiler should by default produce builds for the target
			{Name: "CMAKE_SYSTEM_NAME", Value: "Linux"},                // This will enable cross compiling when explicitly set
			{Name: "CMAKE_C_COMPILER_TARGET", Value: hostTriplet},      // The produced compiler should run on the host (where this build is running), rather than the target
			{Name: "CMAKE_CXX_COMPILER_TARGET", Value: hostTriplet},
			{Name: "CMAKE_C_COMPILER", Value: "clang"},
			{Name: "CMAKE_CXX_COMPILER", Value: "clang++"},
			{Name: "CMAKE_BUILD_TYPE", Value: "Release"},
			{Name: "CMAKE_C_FLAGS", Value: commonFlags},
			{Name: "CMAKE_CXX_FLAGS", Value: commonFlags},
			{Name: "LLVM_ENABLE_PROJECTS", Value: "clang;clang-tools-extra;lld"},
			{Name: "LLVM_ENABLE_RUNTIMES", Value: "compiler-rt;libcxx;libcxxabi;libunwind"},
			{Name: "CMAKE_INSTALL_PREFIX", Value: cb.OutputDirectoryPath},
			{Name: "LLVM_TARGETS_TO_BUILD", Value: "X86"},
			{Name: "LLVM_APPEND_VC_REV", Value: "ON"},
			{Name: "LLVM_ENABLE_PIC", Value: "ON"},                // Required by musl
			{Name: "LLVM_ENABLE_LLD", Value: "ON"},                // Use the LLVM version of ld
			{Name: "LLVM_ENABLE_ZSTD", Value: "FORCE_ON"},         // This is only supported by newer tools, but generally has much better performance than zlib
			{Name: "LLVM_INSTALL_BINUTILS_SYMLINKS", Value: "ON"}, // Increase the change of using LLVM tools even if something is misconfigured somewhere
			{Name: "LLVM_INSTALL_CCTOOLS_SYMLINKS", Value: "ON"},  // Increase the change of using clang even if something is misconfigured somewhere
			{Name: "LLVM_INSTALL_UTILS", Value: "ON"},
			{Name: "LLVM_PARALLEL_LINK_JOBS", Value: fmt.Sprintf("%d", runners.GetCmakeMaxRecommendedParallelLinkJobs())}, // Setting this too high will cause the build processes to get OOM killed
			{Name: "LLVM_CCACHE_BUILD", Value: "ON"},                                                                      // Useful for development to reduce build times
			{Name: "CMAKE_C_COMPILER_LAUNCHER", Value: "ccache"},
			{Name: "CMAKE_CXX_COMPILER_LAUNCHER", Value: "ccache"},
			{Name: "COMPILER_RT_BUILD_SANITIZERS", Value: "OFF"}, // Disable features that are not needed for the cross compiler
			{Name: "CLANG_DEFAULT_RTLIB", Value: "compiler-rt"},  // Use the newly built runtimes
			{Name: "CLANG_DEFAULT_UNWINDLIB", Value: "libunwind"},
			{Name: "CLANG_DEFAULT_CXX_STDLIB", Value: "libc++"},
			// {Name: "LIBCXX_HAS_MUSL_LIBC", Value: "ON"},  // Not sure if this is required or not for the cross compiler
			{Name: "LIBCXX_CXX_ABI", Value: "libcxxabi"}, // Tell the runtimes to link against LLVM libs rather than GCC
			{Name: "LIBCXX_USE_COMPILER_RT", Value: "ON"},
			{Name: "LIBCXXABI_USE_LLVM_UNWINDER", Value: "ON"},
			{Name: "LIBCXXABI_USE_COMPILER_RT", Value: "ON"},
			{Name: "LIBCXXABI_USE_COMPILER_RT", Value: "ON"},
			{Name: "LIBUNWIND_USER_COMPILER_RT", Value: "ON"},
			{Name: "CLANG_VENDOR", Value: cb.Vendor}, // Branding
			{Name: "LLD_VENDOR", Value: cb.Vendor},
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

func (cb *CrossLLVM) recordVersion(buildDirectory string) error {
	cacheVars, err := runners.GetCmakeCacheVars(buildDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to get CMake cache vars from build directory %q", buildDirectory)
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

	cb.builtVersion = fmt.Sprintf("%s.%s.%s", majorVersion, minorVersion, patchVersion)
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

	for _, requiredCommand := range requiredCommands {
		doesExist, _, err := runners.CommandChecker{
			Command: requiredCommand,
		}.DoesCommandExist()
		if err != nil {
			return trace.Wrap(err, "failed to check if command %q exists", requiredCommand)
		}

		if !doesExist {
			return trace.Errorf("required command %q does not exist", requiredCommand)
		}
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
