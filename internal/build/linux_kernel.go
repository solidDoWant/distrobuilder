package build

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gravitational/trace"
	"github.com/otiai10/copy"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type LinuxKernel struct {
	StandardBuilder
	KconfigBuilder
}

func NewLinuxKernel() *LinuxKernel {
	instance := &LinuxKernel{
		StandardBuilder: StandardBuilder{
			Name:            "linux-kernel",
			BinariesToCheck: []string{
				// path.Join("usr", "lib", "libz.so"),
				// path.Join("usr", "bin", "minigzip"),
				// path.Join("bin", "busybox"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (lk *LinuxKernel) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewLinuxGitRepo(repoDirectoryPath, ref)
}

func (lk *LinuxKernel) DoConfiguration(buildDirectoryPath string) error {
	// Copy the source to the build directory. Building out of tree is exceedingly difficult,
	// so build in tree in the build directory.
	err := lk.CopyToBuildDirectory(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to copy source directory %q to build directory %q", lk.SourceDirectoryPath, buildDirectoryPath)
	}

	err = lk.MakeBuild(buildDirectoryPath, nil, "mrproper")
	if err != nil {
		return trace.Wrap(err, "failed to clean the %q", buildDirectoryPath)
	}

	err = lk.CopyKconfig(buildDirectoryPath, nil)
	if err != nil {
		return trace.Wrap(err, "failed to copy the kconfig to %q", buildDirectoryPath)
	}

	return nil
}

func (lk *LinuxKernel) DoBuild(buildDirectoryPath string) error {
	makeOptions, err := lk.getMakeOptions()
	if err != nil {
		return trace.Wrap(err, "failed to build make options")
	}

	_, err = utils.EnsureDirectoryExists(path.Join(lk.OutputDirectoryPath, "boot"))
	if err != nil {
		return trace.Wrap(err, "failed to ensure that the output boot directory exists")
	}

	err = lk.MakeBuild(buildDirectoryPath, makeOptions, "all", "install", "modules_install")
	if err != nil {
		return trace.Wrap(err, "failed to build and install %s", lk.Name)
	}

	err = lk.copySource(makeOptions)
	if err != nil {
		return trace.Wrap(err, "failed to copy source to the build output directory")
	}

	return nil
}

func (lk *LinuxKernel) getMakeOptions() ([]*runners.MakeOptions, error) {
	clangPath, clangxxPath, pkgConfigPath, err := getUtilityPaths()
	if err != nil {
		return nil, trace.Wrap(err, "failed to get clang C and C++ compiler paths")
	}

	// TODO make ccache optional as it can slow down builds under certain circumstances
	return []*runners.MakeOptions{
		{
			Variables: map[string]args.IValue{
				"CC":               args.StringValue("ccache " + lk.GetPathForTool("clang")),
				"CXX":              args.StringValue("ccache " + lk.GetPathForTool("clang++")),
				"NM":               args.StringValue(lk.GetPathForTool("llvm-nm")),
				"AR":               args.StringValue(lk.GetPathForTool("llvm-ar")),
				"LD":               args.StringValue(lk.GetPathForTool("ld.lld")),
				"OBJCOPY":          args.StringValue(lk.GetPathForTool("llvm-objcopy")),
				"OBJDUMP":          args.StringValue(lk.GetPathForTool("llvm-objdump")),
				"READELF":          args.StringValue(lk.GetPathForTool("llvm-readelf")),
				"STRIP":            args.StringValue(lk.GetPathForTool("llvm-strip")),
				"HOSTCC":           args.StringValue("ccache " + clangPath),
				"HOSTCXX":          args.StringValue("ccache " + clangxxPath),
				"PKG_CONFIG":       args.StringValue(pkgConfigPath),
				"LLVM":             args.StringValue("1"),
				"CROSS_COMPILE":    args.StringValue(fmt.Sprintf("%s-", lk.Triplet.String())),
				"ARCH":             args.StringValue(lk.Triplet.Machine),
				"INSTALL_PATH":     args.StringValue(path.Join(lk.OutputDirectoryPath, "boot")),
				"INSTALL_MOD_PATH": args.StringValue(path.Join(lk.OutputDirectoryPath, "usr")),
				"INSTALL_HDR_PATH": args.StringValue(path.Join(lk.OutputDirectoryPath, "usr")),
			},
		},
	}, nil
}

func (lk *LinuxKernel) copySource(makeOptions []*runners.MakeOptions) error {
	kernelVersionOutput, err := runners.Run(&runners.Make{
		GenericRunner: lk.getGenericRunner(lk.SourceDirectoryPath),
		Path:          ".",
		Targets:       []string{"kernelversion"},
		Options:       makeOptions,
	})
	if err != nil {
		return trace.Wrap(err, "failed to get the kernel version")
	}
	kernelVersion := strings.TrimSpace(kernelVersionOutput.Stdout)

	soureDirectoryPath := fmt.Sprintf("linux-%s", kernelVersion)
	relativeKernelSourceDirectoryPath := path.Join("usr", "src", soureDirectoryPath)
	absolutekernelSourceDirectoryPath := path.Join(lk.OutputDirectoryPath, relativeKernelSourceDirectoryPath)
	err = copy.Copy(
		lk.SourceDirectoryPath,
		absolutekernelSourceDirectoryPath,
		copy.Options{
			PreserveTimes:     true,
			OnSymlink:         func(src string) copy.SymlinkAction { return copy.Shallow },
			PermissionControl: copy.PerservePermission,
		},
	)
	if err != nil {
		return trace.Wrap(err, "failed to copy kernel source to %q", absolutekernelSourceDirectoryPath)
	}

	err = filepath.WalkDir(absolutekernelSourceDirectoryPath, func(fsPath string, fsEntry fs.DirEntry, err error) error {
		if err != nil {
			return trace.Wrap(err, "failed to walk dir %q", fsPath)
		}

		err = os.Chown(fsPath, 0, 0)
		if err != nil {
			return trace.Wrap(err, "failed to update owner to root:root on source file %q", fsPath)
		}

		permissions := fs.FileMode(0644)
		if fsEntry.IsDir() {
			permissions = fs.FileMode(0755)
		}

		err = os.Chmod(fsPath, permissions)
		if err != nil {
			return trace.Wrap(err, "failed to update permisions to %o on source file %q", int32(permissions), fsPath)
		}

		return nil
	})

	err = os.Symlink(soureDirectoryPath, path.Join(lk.OutputDirectoryPath, "usr", "src", "linux"))
	if err != nil {
		return trace.Wrap(err, "failed to symlink the generic linux source directory to the specific linux source directory %q ", soureDirectoryPath)
	}

	modulesDirectoryPath := path.Join(lk.OutputDirectoryPath, "usr", "lib", "modules", kernelVersion)
	for _, moduleSubdirectoryPath := range []string{"build", "source"} {
		fullSubdirectoryPath := path.Join(modulesDirectoryPath, moduleSubdirectoryPath)
		err = os.Remove(fullSubdirectoryPath)
		if err != nil {
			return trace.Wrap(err, "failed to remove module subdirectory link %q", fullSubdirectoryPath)
		}

		err = os.Symlink(path.Join(string(os.PathSeparator), relativeKernelSourceDirectoryPath), path.Join(fullSubdirectoryPath))
		if err != nil {
			return trace.Wrap(err, "failed to update module %q symlink to the rootfs-relative source", moduleSubdirectoryPath)
		}

		return nil
	}

	return nil
}
