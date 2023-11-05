package runners

import (
	"fmt"
	"log/slog"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type CommandError struct {
	error
}

func IsCommandError(err error) bool {
	_, ok := err.(CommandError)
	return ok
}

type IRunner interface {
	BuildTask() (*execute.ExecTask, error)
}

type ISetupRunner interface {
	IRunner
	Setup() error
}

type ICleanupRunner interface {
	IRunner
	Cleanup() error
}

func Run(runner IRunner) (*execute.ExecResult, error) {
	var err error
	if cleanupRunner, ok := runner.(ICleanupRunner); ok {
		defer utils.ErrDefer(func() error {
			err := cleanupRunner.Cleanup()
			if err != nil {
				return trace.Wrap(err, "runner cleanup failed")
			}

			return nil
		}, &err)
	}

	if setupRunner, ok := runner.(ISetupRunner); ok {
		err := setupRunner.Setup()
		if err != nil {
			return nil, trace.Wrap(err, "runner setup failed")
		}
	}

	task, err := runner.BuildTask()
	if err != nil || task == nil {
		return nil, trace.Wrap(err, "failed to build task")
	}

	// Debugging
	task.StreamStdio = true

	slog.Debug("running command", "command", prettyPrintTask(task))
	result, err := task.Execute()
	// slog.Debug("command results", "exit_code", result.ExitCode, "stdout", result.Stdout, "stderr", result.Stderr)
	if err != nil {
		return &result, trace.Wrap(err, "failed to execute task: %#v, result: %#v", task, result)
	}
	if result.ExitCode != 0 {
		return &result, CommandError{
			error: trace.Errorf("command failed with exit code %d: %#v", result.ExitCode, result),
		}
	}

	return &result, nil
}

func prettyPrintTask(task *execute.ExecTask) string {
	output := ""

	workingDirectory := task.Cwd
	if workingDirectory == "" {
		workingDirectory = "."
	}
	output += workingDirectory
	output += " $ "

	for _, envVar := range task.Env {
		if envVar == "" {
			continue
		}

		output += envVar
		output += " "
	}

	output += task.Command

	for _, arg := range task.Args {
		if arg == "" {
			continue
		}

		output += fmt.Sprintf(" %q", arg)
	}

	return output
}

func mapToArgs(m map[string]string) []string {
	args := make([]string, 0, len(m))
	for varName, varValue := range m {
		if varName == "" {
			continue
		}

		args = append(args, fmt.Sprintf("%s=%s", varName, varValue))
	}

	return args
}

func JoinMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, _map := range maps {
		for k, v := range _map {
			newVal := result[k]
			if newVal != "" {
				newVal += " "
			}
			newVal += v

			result[k] = newVal
		}
	}

	return result
}
