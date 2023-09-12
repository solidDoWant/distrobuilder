package command_build

import (
	"github.com/urfave/cli/v2"
)

const (
	checkHostRequirementsFlagName string = "check-host-requirements-only"
	skipVerificationFlagName      string = "skip-verification"
)

type Builder interface {
	GetCommand() *cli.Command
}

func BuildCommand() *cli.Command {
	subcommands := []*cli.Command{
		CrossLLVMCommand{}.GetCommand(),
	}

	for _, subcommand := range subcommands {
		processCommand(subcommand)
	}

	return &cli.Command{
		Name:        "build",
		Aliases:     []string{"b"},
		Usage:       "TODO build usage",
		Subcommands: subcommands,
	}
}

// Set values that should apply to all packages/tools
func processCommand(command *cli.Command) {
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

	command.Flags = append(command.Flags, checkHostRequirementsOnlyFlag, skipVerificationFlag)
}
