package build

import (
	"fmt"
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type BusyBox struct {
	StandardBuilder
	KconfigBuilder
}

func NewBusyBox() *BusyBox {
	instance := &BusyBox{
		StandardBuilder: StandardBuilder{
			Name: "busybox",
			BinariesToCheck: []string{
				// path.Join("usr", "lib", "libz.so"),
				// path.Join("usr", "bin", "minigzip"),
				path.Join("bin", "busybox"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (bb *BusyBox) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewBusyBoxGitRepo(repoDirectoryPath, ref)
}

func (bb *BusyBox) DoConfiguration(sourceDirectoryPath string, buildDirectoryPath string) error {
	// Copy the source to the build directory. Building out of tree is exceedingly difficult,
	// so build in tree in the build directory.
	err := bb.CopyToBuildDirectory(sourceDirectoryPath, buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to copy source directory %q to build directory %q", sourceDirectoryPath, buildDirectoryPath)
	}

	// BusyBox should use the same values as a ./configure script, but the
	// exact names are a little different.
	// TODO rethink this architecture and figure out something cleaner
	configureVars := bb.ToolchainRequiredBuilder.GetConfigurenOptions().AdditionalArgs

	replacementVars := map[string]args.IValue{
		"CONFIG_CROSS_COMPILER_PREFIX": args.StringValue(fmt.Sprintf("%s-", bb.Triplet.String())),
		"CONFIG_SYSROOT":               args.StringValue(bb.RootFSDirectoryPath),
		"CONFIG_PREFIX":                args.StringValue(bb.OutputDirectoryPath),
		"CONFIG_EXTRA_CFLAGS":          configureVars["CFLAGS"],
		"CONFIG_EXTRA_LDFLAGS":         configureVars["LIBCC"],
	}
	err = bb.CopyKconfig(buildDirectoryPath, replacementVars)
	if err != nil {
		return trace.Wrap(err, "failed to copy and update the kconfig")
	}

	return nil
}

func (bb *BusyBox) DoBuild(buildDirectoryPath string) error {
	clangPath, clangxxPath, pkgConfigPath, err := getUtilityPaths()
	if err != nil {
		return trace.Wrap(err, "failed to get clang C and C++ compiler paths")
	}

	makeVars := map[string]args.IValue{
		"CC":         args.StringValue(bb.GetPathForTool("clang")),
		"CXX":        args.StringValue(bb.GetPathForTool("clang++")),
		"HOSTCC":     args.StringValue(clangPath),
		"HOSTCXX":    args.StringValue(clangxxPath),
		"PKG_CONFIG": args.StringValue(pkgConfigPath),
	}

	err = bb.MakeBuild(buildDirectoryPath, makeVars, "all", "install")
	if err != nil {
		return trace.Wrap(err, "failed to build and install %s", bb.Name)
	}

	return nil
}

func getUtilityPaths() (string, string, string, error) {
	clangPath, err := utils.SearchPath("clang")
	if err != nil {
		return "", "", "", trace.Wrap(err, "failed to find host clang executable path")
	}

	clangxxPath, err := utils.SearchPath("clang++")
	if err != nil {
		return "", "", "", trace.Wrap(err, "failed to find host clang++ executable path")
	}

	pkgConfigPath, err := utils.SearchPath("pkg-config")
	if err != nil {
		return "", "", "", trace.Wrap(err, "failed to find host pkg-config executable path")
	}

	return clangPath, clangxxPath, pkgConfigPath, nil
}
