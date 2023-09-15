package command_build

import (
	"github.com/urfave/cli/v2"
)

const (
	checkHostRequirementsFlagName string = "check-host-requirements-only"
	skipVerificationFlagName      string = "skip-verification"
	sourceDirectoryPathFlagName   string = "source-directory-path"
	outputDirectoryPathFlagname   string = "output-directory-path"
)

type Builder interface {
	GetCommand() *cli.Command
}

// Builders that use sources should implement this interface
type SourceBuilder interface {
	SetSourcePath(string)
}

type FilesystemOutputBuilder interface {
	SetOutputDirectoryPath(string)
}

func BuildCommand() *cli.Command {
	return &cli.Command{
		Name:        "build",
		Aliases:     []string{"b"},
		Usage:       "TODO build usage",
		Subcommands: getCommands(),
	}
}

func getCommands() []*cli.Command {
	builders := []Builder{
		&CrossLLVMCommand{},
	}

	commands := make([]*cli.Command, 0, len(builders))
	for _, builder := range builders {
		commands = append(commands, getCommand(builder))
	}

	return commands
}

func getCommand(builder Builder) *cli.Command {
	command := builder.GetCommand()

	setCommandFlaags(command, builder)
	command.Action = actionWrapper(builder, command.Action)

	return command
}

func setCommandFlaags(command *cli.Command, builder Builder) {
	checkHostRequirementsOnlyFlag := &cli.BoolFlag{
		Name:    checkHostRequirementsFlagName,
		Usage:   "check host requirements only, do not build",
		Aliases: []string{"c"},
		Value:   false,
	}

	skipVerificationFlag := &cli.BoolFlag{
		Name:    skipVerificationFlagName,
		Usage:   "skip verification checks on the completed build",
		Aliases: []string{"s"},
		Value:   false,
	}

	if _, ok := builder.(SourceBuilder); ok {
		sourceDirectoryPathFlag := &cli.PathFlag{
			Name:    sourceDirectoryPathFlagName,
			Usage:   "directory path that should be used for storing source files",
			Aliases: []string{"S"},
			Value:   "",
		}

		command.Flags = append(command.Flags, sourceDirectoryPathFlag)
	}

	if _, ok := builder.(FilesystemOutputBuilder); ok {
		outputDirectoryPathFlag := &cli.PathFlag{
			Name:    outputDirectoryPathFlagname,
			Usage:   "path where the build outputs should be placed",
			Aliases: []string{"O"},
			Value:   "",
		}

		command.Flags = append(command.Flags, outputDirectoryPathFlag)
	}

	command.Flags = append(command.Flags, checkHostRequirementsOnlyFlag, skipVerificationFlag)
}

func actionWrapper(builder Builder, action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		if sourceBuilder, ok := builder.(SourceBuilder); ok {
			sourceBuilder.SetSourcePath(ctx.Path(sourceDirectoryPathFlagName))
		}

		if sourceBuilder, ok := builder.(FilesystemOutputBuilder); ok {
			sourceBuilder.SetOutputDirectoryPath(ctx.Path(outputDirectoryPathFlagname))
		}

		return action(ctx)
	}
}
