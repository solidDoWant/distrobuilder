package command_build

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/solidDoWant/distrobuilder/internal/utils"
	"github.com/urfave/cli/v2"
)

const (
	gitRefFlagName        string = "git-ref"
	targetTripletFlagName string = "target-triplet"
)

type CrossLLVMCommand struct {
	SourceDirectoryPath string
	OutputDirectoryPath string
}

func (clc *CrossLLVMCommand) GetCommand() *cli.Command {
	return &cli.Command{
		Name: "cross-llvm",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:   gitRefFlagName,
				Usage:  "the fully qualified Git ref to build from",
				Value:  "refs/tags/llvmorg-17.0.0-rc4",
				Action: flags.GitRefValidator,
			},
			&cli.StringFlag{
				Name:   targetTripletFlagName,
				Usage:  "the GNU triplet that the built compiler should target when ran",
				Value:  fmt.Sprintf("%s-pc-linux-musl", clc.getTripletMachineValue()),
				Action: flags.TripletValidator,
			},
		},
		Action: func(cliCtx *cli.Context) error {
			startTime := time.Now()
			builder, err := clc.GetBuilder(cliCtx)
			if err != nil {
				return trace.Wrap(err, "failed to create cross LLVM builder")
			}

			err = builder.CheckHostRequirements()
			if err != nil {
				return trace.Wrap(err, "failed to verify host requirements for cross LLVM builder")
			}

			if cliCtx.Bool(checkHostRequirementsFlagName) {
				slog.Info(fmt.Sprintf("Completed host checks in %v", time.Since(startTime)))
				return nil
			}

			ctx := context.Background() // TODO verify that this is the proper context for this use case
			err = builder.Build(ctx)
			if err != nil {
				return trace.Wrap(err, "failed to build cross LLVM")
			}

			if !cliCtx.Bool(skipVerificationFlagName) {
				err = builder.VerifyBuild(ctx)
				if err != nil {
					return trace.Wrap(err, "failed to verify completed build")
				}
			}

			slog.Info(fmt.Sprintf("Completed build in %v", time.Since(startTime)))

			return nil
		},
	}
}

func (clc *CrossLLVMCommand) GetBuilder(cliCtx *cli.Context) (*build.CrossLLVM, error) {
	targetTriplet, err := utils.ParseTriplet(cliCtx.String(targetTripletFlagName))
	if err != nil {
		return nil, trace.Wrap(err, "failed to parse target triplet")
	}

	builder := build.NewCrossLLVM(targetTriplet)
	builder.GitRef = cliCtx.String(gitRefFlagName)
	builder.SourceDirectoryPath = clc.SourceDirectoryPath
	builder.OutputDirectoryPath = clc.OutputDirectoryPath

	return builder, nil
}

func (clc *CrossLLVMCommand) getTripletMachineValue() string {
	switch runtime.GOARCH {
	case "386":
		return "x86"
	case "amd64":
		return "x86_64"
	default:
		return runtime.GOARCH
	}
}

func (clc *CrossLLVMCommand) SetSourcePath(sourceDirectoryPath string) {
	clc.SourceDirectoryPath = sourceDirectoryPath
}

func (clc *CrossLLVMCommand) SetOutputDirectoryPath(outputDirectoryPath string) {
	clc.OutputDirectoryPath = outputDirectoryPath
}
