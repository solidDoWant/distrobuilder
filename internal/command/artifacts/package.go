package command_artifacts

import (
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/urfave/cli/v2"
)

const (
	buildOutputPathFlagName string = "build-output-path"
	outputFilePathFlagName  string = "output-file-path"
)

type Packager interface {
	GetPackageCommand() *cli.Command
}

type FilesystemOutputPackager interface {
	SetOutputFilePath(string)
}

func PackageCommand() *cli.Command {
	return &cli.Command{
		Name:        "package",
		Aliases:     []string{"p"},
		Usage:       "TODO package usage",
		Subcommands: getCommands(),
	}
}

func getCommands() []*cli.Command {
	packagers := []Packager{
		&TarballCommand{},
	}

	commands := make([]*cli.Command, 0, len(packagers))
	for _, packager := range packagers {
		commands = append(commands, getCommand(packager))
	}

	return commands
}

func getCommand(packager Packager) *cli.Command {
	command := packager.GetPackageCommand()

	setCommandFlaags(command, packager)
	command.Action = actionWrapper(packager, command.Action)

	return command
}

func setCommandFlaags(command *cli.Command, packager Packager) {
	buildOutputPathFlag := &cli.PathFlag{
		Name:     buildOutputPathFlagName,
		Usage:    "path to the build output directory",
		Aliases:  []string{"s"},
		Required: true,
		Action:   flags.ExistingDirValidator,
	}

	command.Flags = append(command.Flags, buildOutputPathFlag)

	if _, ok := packager.(FilesystemOutputPackager); ok {
		buildOutputPathFlag := &cli.PathFlag{
			Name:     outputFilePathFlagName,
			Usage:    "path to the package output file",
			Aliases:  []string{"O"},
			Required: false,
		}

		command.Flags = append(command.Flags, buildOutputPathFlag)
	}

}

func actionWrapper(packager Packager, action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		if filesystemOutputPackager, ok := packager.(FilesystemOutputPackager); ok {
			filesystemOutputPackager.SetOutputFilePath(ctx.Path(outputFilePathFlagName))
		}

		return action(ctx)
	}
}
