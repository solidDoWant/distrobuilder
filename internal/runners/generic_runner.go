package runners

import (
	"fmt"

	execute "github.com/alexellis/go-execute/pkg/v1"
	pie "github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

func MergeGenericRunnerOptions(options ...*GenericRunnerOptions) (*GenericRunnerOptions, error) {
	options = utils.FilterNil(options)
	mergedEnvironmentVariables, err := args.MergeMap(pie.Map(options, func(option *GenericRunnerOptions) map[string]args.IValue { return option.EnvironmentVariables })...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge all enviroment variables")
	}

	return &GenericRunnerOptions{
		EnvironmentVariables: mergedEnvironmentVariables,
	}, nil
}

type GenericRunnerOptions struct {
	EnvironmentVariables map[string]args.IValue
}

type GenericRunner struct {
	WorkingDirectory string
	Options          []*GenericRunnerOptions
}

func (gr GenericRunner) BuildTask() (*execute.ExecTask, error) {
	mergedOptions, err := MergeGenericRunnerOptions(gr.Options...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge generic runner options")
	}

	// TODO consider replacing this library with something that allows for piping the command output to slog in real time
	return &execute.ExecTask{
		Env: pie.Map(pie.Keys(mergedOptions.EnvironmentVariables), func(varName string) string {
			return fmt.Sprintf("%s=%s", varName, mergedOptions.EnvironmentVariables[varName].GetValue())
		}),
		Cwd: gr.WorkingDirectory,
	}, nil
}
