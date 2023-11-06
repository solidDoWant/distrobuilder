package build

import (
	"os"
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type PCRE2 struct {
	StandardBuilder
}

func NewPCRE2() *PCRE2 {
	instance := &PCRE2{
		StandardBuilder: StandardBuilder{
			Name: "PCRE2",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "pcre2grep"),
				path.Join("usr", "bin", "pcre2test"),
				path.Join("usr", "lib", "libpcre2-8.so"),
				path.Join("usr", "lib", "libpcre2-16.so"),
				path.Join("usr", "lib", "libpcre2-32.so"),
				path.Join("usr", "lib", "libpcre2-posix.so"),
			},
		},
	}

	// There is not currently Golang syntactic sugar for this pattern
	instance.IStandardBuilder = instance
	return instance
}

func (pcre2 *PCRE2) GetGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	return git_source.NewPCRE2GitRepo(repoDirectoryPath, ref)
}

func (pcre2 *PCRE2) DoConfiguration(buildDirectoryPath string) error {
	return pcre2.AutogenConfigure(buildDirectoryPath,
		"--enable-pcre2-16",
		"--enable-pcre2-32",
		"--enable-jit=auto",
		"--enable-jit-sealloc",
		"--enable-newline-is-any", // TODO not sure that this should be set
		"--enable-unicode",
		"--enable-pcre2grep-libz",
		// "--enable-pcre2grep-libbz2",	// TODO enable this once bz2 lib has been added
		// "--enable-pcre2test-libedit",	// TODO consider enabling this if libedit is added
	)
}

func (pcre2 *PCRE2) DoBuild(buildDirectoryPath string) error {
	err := pcre2.MakeBuild(buildDirectoryPath, pcre2.getMakeOptions(), "install")
	if err != nil {
		return trace.Wrap(err, "failed to perform make install on %q", buildDirectoryPath)
	}

	err = pcre2.UpdatePkgconfigsPrefixes(pcre2.OutputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to update package config files")
	}

	err = pcre2.updateConfigScript()
	if err != nil {
		return trace.Wrap(err, "failed to update config script with appropriate prefix value")
	}

	err = pcre2.RunLibtool(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run libtool on the build output")
	}

	return nil
}

func (pcre2 *PCRE2) updateConfigScript() error {
	scriptPath := path.Join(pcre2.OutputDirectoryPath, "usr", "bin", "pcre2-config")
	lines, err := utils.ReadLines(scriptPath)
	if err != nil {
		return trace.Wrap(err, "failed to read the script file %q lines", scriptPath)
	}

	fileHandle, err := os.OpenFile(scriptPath, os.O_TRUNC|os.O_WRONLY, 0)
	defer utils.Close(fileHandle, &err)
	if err != nil {
		return trace.Wrap(err, "failed to open script file %q for writing", scriptPath)
	}

	for _, line := range lines {
		if line == "prefix=/" {
			line = "prefix=/usr"
		}

		_, err = fileHandle.WriteString(line + "\n")
		if err != nil {
			return trace.Wrap(err, "failed to write line %q to script file %q", line, scriptPath)
		}
	}

	return nil
}

func (pcre2 *PCRE2) getMakeOptions() []*runners.MakeOptions {
	return []*runners.MakeOptions{
		{
			Variables: map[string]args.IValue{
				"DESTDIR": args.StringValue(path.Join(pcre2.OutputDirectoryPath, "usr")),
			},
		},
	}
}
