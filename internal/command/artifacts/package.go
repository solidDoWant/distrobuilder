package command_artifacts

import (
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/urfave/cli/v2"
)

const buildOutputPathFlagName string = "build-output-path"

func PackageCommand() *cli.Command {
	subcommands := []*cli.Command{
		TarballCommand{}.GetPackageCommand(),
	}

	for _, subcommand := range subcommands {
		processPackageCommand(subcommand)
	}

	return &cli.Command{
		Name:        "package",
		Aliases:     []string{"p"},
		Usage:       "TODO package usage",
		Subcommands: subcommands,
	}
}

// Set values that should apply to all packages/tools
func processPackageCommand(command *cli.Command) {
	buildOutputPathFlag := &cli.PathFlag{
		Name:     buildOutputPathFlagName,
		Usage:    "path to the build output directory",
		Aliases:  []string{"s"},
		Required: true,
		Action:   flags.ExistingDirValidator,
	}

	command.Flags = append(command.Flags, buildOutputPathFlag)
}
