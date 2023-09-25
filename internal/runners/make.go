package runners

import (
	"fmt"
	"runtime"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
)

type Make struct {
	GenericRunner
	Path      string // Path to the directory containing the makefile, relative to the working directory
	Targets   []string
	Variables map[string]args.IValue
}

func (m *Make) BuildTask() (*execute.ExecTask, error) {
	task, err := m.GenericRunner.BuildTask()
	if err != nil {
		return task, trace.Wrap(err, "failed to create generic runner task")
	}

	args, err := m.buildArgs()
	if err != nil {
		return nil, trace.Wrap(err, "failed to create runner args")
	}
	args = append(
		args,
		"--no-print-directory",                // This makes outputs hard to parse when not set
		fmt.Sprintf("-j%d", runtime.NumCPU()), // Use all CPU cores
	)
	task.Args = append(task.Args, args...)
	task.Command = "make"

	return task, nil
}

func (m *Make) buildArgs() ([]string, error) {
	args := []string{}

	if m.Path != "" {
		args = append(args, "-C", m.Path)
	}

	for _, target := range m.Targets {
		args = append(args, target)
	}

	for variableName, variableValue := range m.Variables {
		args = append(args, fmt.Sprintf("%s=%s", variableName, variableValue.GetValue()))
	}

	return args, nil
}
