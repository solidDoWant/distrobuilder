package command_build

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/urfave/cli/v2"
)

const (
	checkHostRequirementsFlagName string = "check-host-requirements-only"
	skipVerificationFlagName      string = "skip-verification"
	sourceDirectoryPathFlagName   string = "source-directory-path"
	outputDirectoryPathFlagname   string = "output-directory-path"
	gitRefFlagName                string = "git-ref"
)

type Builder interface {
	GetCommand() *cli.Command
}

type DefaultActionBuilder interface {
	GetBuilder(cliCtx *cli.Context) (build.Builder, error)
}

// Builders that use sources should implement this interface
type SourceBuilder interface {
	SetSourcePath(string)
}

type FilesystemOutputBuilder interface {
	SetOutputDirectoryPath(string)
	GetOutputDirectoryPath() string
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
		&LinuxHeadersCommand{},
	}

	commands := make([]*cli.Command, 0, len(builders))
	for _, builder := range builders {
		commands = append(commands, getCommand(builder))
	}

	return commands
}

func getCommand(builder Builder) *cli.Command {
	command := builder.GetCommand()

	setCommandFlags(command, builder)

	action := command.Action
	if defaultActionBuilder, ok := builder.(DefaultActionBuilder); ok && action == nil {
		action = builderAction(defaultActionBuilder)
	}

	command.Action = actionWrapper(builder, action)

	return command
}

func setCommandFlags(command *cli.Command, builder Builder) {
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

		if outputBuilder, ok := builder.(FilesystemOutputBuilder); ok {
			outputBuilder.SetOutputDirectoryPath(ctx.Path(outputDirectoryPathFlagname))
		}

		return action(ctx)
	}
}

func builderAction(actionBuilder DefaultActionBuilder) cli.ActionFunc {
	action := func(cliCtx *cli.Context) error {
		startTime := time.Now()
		builder, err := actionBuilder.GetBuilder(cliCtx)
		if err != nil {
			return trace.Wrap(err, "failed to create builder")
		}

		err = builder.CheckHostRequirements()
		if err != nil {
			return trace.Wrap(err, "failed to verify host requirements for builder")
		}

		if cliCtx.Bool(checkHostRequirementsFlagName) {
			slog.Info(fmt.Sprintf("Completed host checks in %v", time.Since(startTime)))
			return nil
		}

		ctx := context.Background() // TODO verify that this is the proper context for this use case
		err = builder.Build(ctx)
		if err != nil {
			return trace.Wrap(err, "build failed")
		}

		if !cliCtx.Bool(skipVerificationFlagName) {
			err = builder.VerifyBuild(ctx)
			if err != nil {
				return trace.Wrap(err, "failed to verify completed build")
			}
		}

		args := make([]any, 0, 2) // slog.Info requires "any" as the type
		if outputBuilder, ok := actionBuilder.(FilesystemOutputBuilder); ok {
			args = append(args, "output_directory", outputBuilder.GetOutputDirectoryPath())
		}

		slog.Info(fmt.Sprintf("Completed build in %v", time.Since(startTime)), args...)

		return nil
	}

	return action
}
