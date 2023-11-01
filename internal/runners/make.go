package runners

import (
	"fmt"
	"runtime"

	execute "github.com/alexellis/go-execute/pkg/v1"
	pie "github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type MakeOptions struct {
	Variables map[string]args.IValue
}

func MergeMakeOptions(options ...*MakeOptions) (*MakeOptions, error) {
	options = utils.FilterNil(options)
	mergeVariables, err := args.MergeMap(pie.Map(options, func(option *MakeOptions) map[string]args.IValue { return option.Variables })...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge all Makefile variable values")
	}

	return &MakeOptions{
		Variables: mergeVariables,
	}, nil
}

type Make struct {
	GenericRunner
	Path    string // Path to the directory containing the makefile, relative to the working directory
	Targets []string
	Options []*MakeOptions
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

	mergedOptions, err := MergeMakeOptions(m.Options...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge CMake options")
	}

	for variableName, variableValue := range mergedOptions.Variables {
		args = append(args, fmt.Sprintf("%s=%s", variableName, variableValue.GetValue()))
	}

	return args, nil
}
