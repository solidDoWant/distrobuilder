package runners

import (
	"fmt"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
)

type EnvironmentVariable struct {
	Name  string
	Value string
}

func (ev *EnvironmentVariable) AsString() (string, error) {
	if ev.Value == "" {
		return "", trace.Errorf("variable name must be set")
	}

	if ev.Value == "" {
		return "", trace.Errorf("variable value must be set")
	}

	return fmt.Sprintf("%s=%q", ev.Name, ev.Value), nil
}

type GenericRunner struct {
	EnvironmentVariables []EnvironmentVariable
	WorkingDirectory     string
}

func (gr GenericRunner) BuildTask() (*execute.ExecTask, error) {
	envVars := make([]string, 0, len(gr.EnvironmentVariables))
	for _, environmentVariable := range gr.EnvironmentVariables {
		convertedEnvironmentVariable, err := environmentVariable.AsString()
		if err != nil {
			return nil, trace.Wrap(err, "failed to convert environment variable %v to string", environmentVariable)
		}

		envVars = append(envVars, convertedEnvironmentVariable)
	}

	// TODO consider replacing this library with something that llows for piping the command output to slog in real time
	return &execute.ExecTask{
		Env: envVars,
		Cwd: gr.WorkingDirectory,
	}, nil
}
