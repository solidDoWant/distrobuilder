package build

import (
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

// This builder is currently bugged. The `iconv` binary built includes a
// rpath that points to the output directory, even with `--disable-rpath`
// set. I've filed a bug report upstream and am awaiting a fix.

type Libiconv struct {
	StandardBuilder
}

func NewLibiconv() *Libiconv {
	instance := &Libiconv{
		StandardBuilder: StandardBuilder{
			Name: "libiconv",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "iconv"),
				path.Join("usr", "lib", "libcharset.so"),
				path.Join("usr", "lib", "libiconv.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (l *Libiconv) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewLibiconvGitRepo(repoDirectoryPath, ref)
}

func (l *Libiconv) DoConfiguration(buildDirectoryPath string) error {
	err := l.CopyToBuildDirectory(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to copy sources to build directory")
	}

	_, err = runners.Run(
		runners.CommandRunner{
			GenericRunner: runners.GenericRunner{
				WorkingDirectory: buildDirectoryPath,
			},
			Command: path.Join(buildDirectoryPath, "autopull.sh"),
		},
	)
	if err != nil {
		return trace.Wrap(err, "failed to run autopull script")
	}

	_, err = runners.Run(
		runners.CommandRunner{
			GenericRunner: runners.GenericRunner{
				WorkingDirectory: buildDirectoryPath,
				Options: []*runners.GenericRunnerOptions{
					{
						EnvironmentVariables: map[string]args.IValue{
							// This script runs tools on the host that need to be compiled
							"CC": args.StringValue("clang"),
						},
					},
				},
			},
			Command: path.Join(buildDirectoryPath, "autogen.sh"),
		},
	)
	if err != nil {
		return trace.Wrap(err, "failed to run autogen script")
	}

	err = l.GNUConfigureWithSrc(buildDirectoryPath, buildDirectoryPath,
		"--enable-static",
		"--enable-extra-encodings",
		"--enable-year2038",
		"--disable-rpath",
	)
	if err != nil {
		return trace.Wrap(err, "failed to configure project")
	}

	return nil
}

func (l *Libiconv) DoBuild(buildDirectoryPath string) error {
	err := l.MakeBuild(buildDirectoryPath, l.getMakeOptions(), "install-strip")
	if err != nil {
		return trace.Wrap(err, "failed to perform make build on %s", l.Name)
	}

	err = l.RunLibtool(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run libtool on the build output")
	}

	return nil
}

func (l *Libiconv) getMakeOptions() []*runners.MakeOptions {
	return []*runners.MakeOptions{
		{
			Variables: map[string]args.IValue{
				"DESTDIR": args.StringValue(path.Join(l.OutputDirectoryPath, "usr")),
			},
		},
	}
}
