package command_build

import (
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/urfave/cli/v2"
)

const linuxGitRefFlagName string = "git-ref"

type LinuxHeadersCommand struct {
	SourceDirectoryPath string
	OutputDirectoryPath string
}

func (lh *LinuxHeadersCommand) GetCommand() *cli.Command {
	return &cli.Command{
		Name: "linux-headers",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:   linuxGitRefFlagName,
				Usage:  "the fully qualified Git ref to build the linux headers from",
				Value:  "refs/tags/v6.5",
				Action: flags.GitRefValidator,
			},
		},
	}
}

func (lh *LinuxHeadersCommand) GetBuilder(cliCtx *cli.Context) (build.Builder, error) {
	builder := &build.LinuxHeaders{
		SourceBuilder: build.SourceBuilder{
			SourceDirectoryPath: lh.SourceDirectoryPath,
		},
		FilesystemOutputBuilder: build.FilesystemOutputBuilder{
			OutputDirectoryPath: lh.OutputDirectoryPath,
		},
		GitRef: cliCtx.String(linuxGitRefFlagName),
	}

	return builder, nil
}

func (lh *LinuxHeadersCommand) SetSourcePath(sourceDirectoryPath string) {
	lh.SourceDirectoryPath = sourceDirectoryPath
}

func (lh *LinuxHeadersCommand) SetOutputDirectoryPath(outputDirectoryPath string) {
	lh.OutputDirectoryPath = outputDirectoryPath
}

func (lh *LinuxHeadersCommand) GetOutputDirectoryPath() string {
	return lh.OutputDirectoryPath
}
