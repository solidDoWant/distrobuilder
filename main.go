package main

import (
	"log/slog"
	"os"

	"github.com/gravitational/trace"
	command_artifacts "github.com/solidDoWant/distrobuilder/internal/command/artifacts"
	command_build "github.com/solidDoWant/distrobuilder/internal/command/build"
	"github.com/urfave/cli/v2"
)

func main() {
	configureLogger()

	app := &cli.App{
		Name:                 "distrobuilder",
		Usage:                "TODO global usage",
		Version:              "v0.0.1",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			command_build.BuildCommand(),
			command_artifacts.PackageCommand(),
			command_artifacts.InstallCommand(),
		},
		// TODO allow for setting log level
	}

	err := app.Run(os.Args)
	if err != nil {
		exitHandler(trace.Wrap(err, "an error occured while runnning the app"))
	}
}

func exitHandler(err error) {
	if err == nil {
		os.Exit(0)
	}

	exitCode := 1
	if castedError, ok := err.(cli.ExitCoder); ok {
		exitCode = castedError.ExitCode()
	}

	slog.Error(err.Error())
	os.Exit(exitCode)
}

func configureLogger() {
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))
}
