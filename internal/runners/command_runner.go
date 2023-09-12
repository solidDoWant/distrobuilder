package runners

import (
	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
)

type CommandRunner struct {
	GenericRunner
	Command   string // Can be fully qualified path, relative path, or binary name. If binary name then it will be searched for via the $PATH environment variable.
	Arguments []string
}

func (cr CommandRunner) BuildTask() (*execute.ExecTask, error) {
	task, err := cr.GenericRunner.BuildTask()
	if err != nil {
		return task, trace.Wrap(err, "failed to create generic runner task")
	}

	if cr.Command == "" {
		return nil, trace.Errorf("command was not provided")
	}
	task.Command = cr.Command

	args := make([]string, 0, len(cr.Arguments))
	for _, arg := range cr.Arguments {
		if arg == "" {
			continue
		}

		args = append(args, arg)
	}
	task.Args = args

	return task, nil
}

func (cr *CommandRunner) PrettyPrint() string {
	output := ""
	output += cr.Command

	for _, argument := range cr.Arguments {
		if argument == "" {
			continue
		}

		output += " "
		output += argument
	}

	return output
}
