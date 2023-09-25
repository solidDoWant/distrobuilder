package build

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
)

type MuslLibc struct {
	StandardBuilder

	// Vars for validation checking
	sourceVersion string
}

func NewMuslLibc() *MuslLibc {
	instance := &MuslLibc{
		StandardBuilder: StandardBuilder{
			Name: "musl-libc",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "lz4"),
				path.Join("usr", "lib", "liblz4.so"),
			},
		},
	}

	// There is not currently Golang syntactic sugar for this pattern
	instance.IStandardBuilder = instance
	return instance
}

func (ml *MuslLibc) GetGitRepo(repoDirectoryPath, ref string) *source.GitRepo {
	return git_source.NewMuslGitRepo(repoDirectoryPath, ref)
}

func (ml *MuslLibc) DoConfiguration(sourceDirectoryPath, buildDirectoryPath string) error {
	err := ml.GNUConfigure(sourceDirectoryPath, buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to build %s", ml.Name)
	}

	// Record values for build verification
	versionFilePath := path.Join(sourceDirectoryPath, "VERSION")
	fileContents, err := os.ReadFile(versionFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to read VERSION file at %q", versionFilePath)
	}
	ml.sourceVersion = strings.TrimSpace(string(fileContents))

	return nil
}

func (ml *MuslLibc) DoBuild(buildDirectoryPath string) error {
	return ml.MakeBuild(buildDirectoryPath, map[string]args.IValue{"DESTDIR": args.StringValue(path.Join(ml.OutputDirectoryPath, "usr"))}, "install")
}

func (ml *MuslLibc) VerifyBuild(ctx context.Context) error {
	libcPath := path.Join(ml.OutputDirectoryPath, "usr", "lib", "libc.so")
	isValid, version, err := (&runners.VersionChecker{
		CommandRunner: runners.CommandRunner{
			Command: libcPath,
			Arguments: []string{
				"--version",
			},
		},
		IgnoreErrorExit: true,
		UseStdErr:       true,
		VersionRegex:    fmt.Sprintf("(?m)^Version %s$", runners.SemverRegex),
		VersionChecker:  runners.ExactSemverChecker(ml.sourceVersion),
	}).IsValidVersion()
	if err != nil {
		return trace.Wrap(err, "failed to retreive built musl libc version")
	}

	if !isValid {
		return trace.Errorf("built musl libc version %q does not match build version %q", version, ml.sourceVersion)
	}

	err = ml.VerifyTargetElfFile(libcPath)
	if err != nil {
		return trace.Wrap(err, "libc file %q did not match the expected ELF values", libcPath)
	}

	return nil
}
