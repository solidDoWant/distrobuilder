package runners

import execute "github.com/alexellis/go-execute/pkg/v1"

type EnvironmentVariables map[string]string

func (evs *EnvironmentVariables) AsArgs() []string {
	return mapToArgs(*evs)
}

type GenericRunner struct {
	EnvironmentVariables EnvironmentVariables
	WorkingDirectory     string
}

func (gr GenericRunner) BuildTask() (*execute.ExecTask, error) {
	// TODO consider replacing this library with something that allows for piping the command output to slog in real time
	return &execute.ExecTask{
		Env: gr.EnvironmentVariables.AsArgs(),
		Cwd: gr.WorkingDirectory,
	}, nil
}
