package runners

import (
	"fmt"
	"log/slog"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
)

type CommandError struct {
	error
}

func IsCommandError(err error) bool {
	_, ok := err.(CommandError)
	return ok
}

type Runner interface {
	BuildTask() (*execute.ExecTask, error)
}

func Run(runner Runner) (*execute.ExecResult, error) {
	task, err := runner.BuildTask()
	if err != nil || task == nil {
		return nil, trace.Wrap(err, "failed to build task")
	}

	slog.Debug("running command", "command", prettyPrintTask(task))
	result, err := task.Execute()
	slog.Debug("command results", "exit_code", result.ExitCode, "stdout", result.Stdout, "stderr", result.Stderr)
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

		output += " "
		output += arg
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
