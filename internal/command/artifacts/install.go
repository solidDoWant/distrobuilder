package command_artifacts

import (
	"github.com/solidDoWant/distrobuilder/internal/artifacts"
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/urfave/cli/v2"
)

const (
	sourcePathFlagName  = "source-path"
	installPathFlagName = "install-path"
)

func InstallCommand() *cli.Command {
	subcommands := []*cli.Command{
		TarballCommand{}.GetInstallCommand(),
	}

	for _, subcommand := range subcommands {
		processInstallCommand(subcommand)
	}

	return &cli.Command{
		Name:        "install",
		Aliases:     []string{"i"},
		Usage:       "TODO install usage",
		Subcommands: subcommands,
	}
}

func processInstallCommand(command *cli.Command) {
	sourcePathFlag := &cli.PathFlag{
		Name:     sourcePathFlagName,
		Usage:    "path to the package file",
		Aliases:  []string{"s"},
		Required: true,
		Action:   flags.ExistingFileValidator,
	}

	installPathFlag := &cli.PathFlag{
		Name:     installPathFlagName,
		Usage:    "path to the directory where the package will be installed",
		Aliases:  []string{"D"},
		Required: true,
	}

	command.Flags = append(command.Flags, sourcePathFlag, installPathFlag)
}

func getArtifactInstallOptions(cliCtx *cli.Context) *artifacts.InstallOptions {
	return &artifacts.InstallOptions{
		SourcePath:  cliCtx.Path(sourcePathFlagName),
		InstallPath: cliCtx.Path(installPathFlagName),
	}
}
