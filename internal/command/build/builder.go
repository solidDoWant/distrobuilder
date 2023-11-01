package command_build

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/solidDoWant/distrobuilder/internal/utils"
	"github.com/urfave/cli/v2"
)

const (
	checkHostRequirementsFlagName string = "check-host-requirements-only"
	skipVerificationFlagName      string = "skip-verification"
)

type Builder interface {
	GetCommand() *cli.Command
	GetBuilder(cliCtx *cli.Context) (build.IBuilder, error)
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
		&RootFilesystemCommand{},
		&LinuxHeadersCommand{},
		NewMuslLibcCommand(),
		NewZlibNgCommand(),
		NewXZCommand(),
		NewLZ4Command(),
		NewZstdCommand(),
		NewBusyBoxCommand(),
		NewLibreSSLCommand(),
		NewLinuxKernelCommand(),
		NewFreeTypeCommand(),
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

	if command.Action == nil {
		command.Action = builderAction(builder)
	}

	return command
}

// These flags are supported by all builders by virtue of the interface
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

	command.Flags = append(command.Flags, checkHostRequirementsOnlyFlag, skipVerificationFlag)
}
func builderAction(builder Builder) cli.ActionFunc {
	action := func(cliCtx *cli.Context) error {
		startTime := time.Now()
		builder, err := builder.GetBuilder(cliCtx)
		if err != nil {
			return trace.Wrap(err, "failed to create builder")
		}

		setValuesForInterfaceFlags(builder, cliCtx)

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
		if outputBuilder, ok := builder.(build.IFilesystemOutputBuilder); ok {
			args = append(args, "output_directory", outputBuilder.GetOutputDirectoryPath())
		}

		slog.Info(fmt.Sprintf("Completed build in %v", time.Since(startTime)), args...)

		return nil
	}

	return action
}

// Transfers flags for optional interfaces from the command to the builder
// This function should be called during a command's action
func setValuesForInterfaceFlags(builder build.IBuilder, cliCtx *cli.Context) {
	if sourceBuilder, ok := builder.(build.ISourceBuilder); ok {
		sourceBuilder.SetSourceDirectoryPath(cliCtx.Path(sourceDirectoryPathFlag.Name))
	}

	if outputBuilder, ok := builder.(build.IFilesystemOutputBuilder); ok {
		outputBuilder.SetOutputDirectoryPath(cliCtx.Path(outputDirectoryPathFlag.Name))
	}

	if targetTripletBuilder, ok := builder.(build.ITargetTripletBuilder); ok {
		// This is validated by the CLI parser so it is safe to throw away the error ret value here
		triplet, _ := utils.ParseTriplet(cliCtx.String(targetTripletFlag.Name))
		targetTripletBuilder.SetTargetTriplet(triplet)
	}

	if toolchainBuilder, ok := builder.(build.IToolchainRequiredBuilder); ok {
		toolchainBuilder.SetToolchainDirectory(cliCtx.Path(toolchainDirectoryPathFlag.Name))
	}

	if gitRefBuilder, ok := builder.(build.IGitRefBuilder); ok {
		gitRefBuilder.SetGitRef(cliCtx.String(gitRefFlag.Name))
	}

	if rootFSBuilder, ok := builder.(build.IRootFSBuilder); ok {
		rootFSBuilder.SetRootFSDirectoryPath(cliCtx.Path(rootFSDirectoryPathFlag.Name))
	}

	if kconfigBuilder, ok := builder.(build.IKconfigBuilder); ok {
		kconfigBuilder.SetConfigFilePath(cliCtx.Path(configPathFlag.Name))
	}
}
