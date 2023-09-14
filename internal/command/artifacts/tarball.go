package command_artifacts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/artifacts"
	"github.com/urfave/cli/v2"
)

const clearOwnerFlagName = "clear-owner"

type TarballCommand struct {
	OutputFilePath string
}

func (tc *TarballCommand) GetPackageCommand() *cli.Command {
	return &cli.Command{
		Name:  "tarball",
		Usage: "Packages a build into a tarball",
		Flags: tc.getCommonFlags(),
		Action: func(cliCtx *cli.Context) error {
			startTime := time.Now()
			packager, err := tc.GetArtifactHandler(cliCtx)
			if err != nil {
				return trace.Wrap(err, "failed to create tarball packager")
			}

			ctx := context.Background() // TODO verify that this is the proper context for this use case
			_, err = packager.Package(ctx)
			if err != nil {
				return trace.Wrap(err, "failed to create tarball package")
			}

			slog.Info(fmt.Sprintf("Created package in %v", time.Since(startTime)))
			return nil
		},
	}
}

func (tc TarballCommand) GetInstallCommand() *cli.Command {
	return &cli.Command{
		Name:  "tarball",
		Usage: "Installs a build from a tarball",
		Flags: tc.getCommonFlags(),
		Action: func(cliCtx *cli.Context) error {
			startTime := time.Now()
			packager, err := tc.GetArtifactHandler(cliCtx)
			if err != nil {
				return trace.Wrap(err, "failed to create tarball installer")
			}

			ctx := context.Background() // TODO verify that this is the proper context for this use case
			err = packager.Install(ctx, getArtifactInstallOptions(cliCtx))
			if err != nil {
				return trace.Wrap(err, "failed to install tarball package")
			}

			slog.Info(fmt.Sprintf("Installed package in %v", time.Since(startTime)))
			return nil
		},
	}
}

func (tc *TarballCommand) getCommonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    clearOwnerFlagName,
			Usage:   "Clear owner and group of tarball",
			Aliases: []string{"c"},
			Value:   false,
		},
	}
}

func (tc *TarballCommand) GetArtifactHandler(cliCtx *cli.Context) (*artifacts.Tarball, error) {
	tarballPackager := &artifacts.Tarball{}
	tarballPackager.ShouldResetOwner = cliCtx.Bool(clearOwnerFlagName)
	tarballPackager.SourcePath = cliCtx.Path(buildOutputPathFlagName)
	tarballPackager.SetOutputFilePath(tc.OutputFilePath)

	return tarballPackager, nil
}

func (tc *TarballCommand) SetOutputFilePath(outputFilePath string) {
	tc.OutputFilePath = outputFilePath
}
