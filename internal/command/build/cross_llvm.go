package command_build

import (
	"fmt"
	"runtime"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/build"
	"github.com/solidDoWant/distrobuilder/internal/command/flags"
	"github.com/solidDoWant/distrobuilder/internal/utils"
	"github.com/urfave/cli/v2"
)

const (
	muslGitRefFlagName    string = "musl-" + gitRefFlagName
	llvmGitRefFlagName    string = "llvm-" + gitRefFlagName
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
				Name:   muslGitRefFlagName,
				Usage:  "the fully qualified Git ref to build Musl from",
				Value:  "refs/tags/v1.2.4",
				Action: flags.GitRefValidator,
			},
			&cli.StringFlag{
				Name:   llvmGitRefFlagName,
				Usage:  "the fully qualified Git ref to build LLVM from",
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
	}
}

func (clc *CrossLLVMCommand) GetBuilder(cliCtx *cli.Context) (build.Builder, error) {
	targetTriplet, err := utils.ParseTriplet(cliCtx.String(targetTripletFlagName))
	if err != nil {
		return nil, trace.Wrap(err, "failed to parse target triplet")
	}

	builder := build.NewCrossLLVM(targetTriplet)
	builder.LLVMGitRef = cliCtx.String(llvmGitRefFlagName)
	builder.MuslGitRef = cliCtx.String(muslGitRefFlagName)
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

func (clc *CrossLLVMCommand) GetOutputDirectoryPath() string {
	return clc.OutputDirectoryPath
}
