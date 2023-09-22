package runners

import (
	"fmt"

	execute "github.com/alexellis/go-execute/pkg/v1"
	pie "github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type ConfigureOptions struct {
	AdditionalArgs  map[string]args.IValue
	AdditionalFlags []args.IValue
}

func MergeConfigurationOptions(options ...*ConfigureOptions) (*ConfigureOptions, error) {
	options = utils.FilterNil(options)
	mergedAdditionalArgs, err := args.MergeMap(pie.Map(options, func(option *ConfigureOptions) map[string]args.IValue { return option.AdditionalArgs })...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge additional args from all options")
	}

	return &ConfigureOptions{
		AdditionalArgs:  mergedAdditionalArgs,
		AdditionalFlags: utils.DedupeReduce(pie.Map(options, func(option *ConfigureOptions) []args.IValue { return option.AdditionalFlags })...),
	}, nil
}

type Configure struct {
	GenericRunner
	ConfigurePath string
	HostTriplet   *utils.Triplet
	TargetTriplet *utils.Triplet
	Options       []*ConfigureOptions
}

func (c *Configure) BuildTask() (*execute.ExecTask, error) {
	task, err := c.GenericRunner.BuildTask()
	if err != nil {
		return task, trace.Wrap(err, "failed to create generic runner task")
	}

	args, err := c.buildArgs()
	if err != nil {
		return nil, trace.Wrap(err, "failed to create runner args")
	}

	task.Args = append(task.Args, args...)
	task.Command = c.ConfigurePath

	return task, nil
}

func (c *Configure) buildArgs() ([]string, error) {
	mergedOptions, err := MergeConfigurationOptions(c.Options...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge all configuration options")
	}

	args := pie.Flat([][]string{
		c.buildTripletArgs(),
		pie.Map(mergedOptions.AdditionalFlags, func(v args.IValue) string { return v.GetValue() }),
		pie.Map(pie.Keys(mergedOptions.AdditionalArgs), func(varName string) string {
			return fmt.Sprintf("%s=%s", varName, mergedOptions.AdditionalArgs[varName].GetValue())
		}),
	})

	return args, nil
}

func (c *Configure) buildTripletArgs() []string {
	args := []string{}

	hostTripletArg := c.buildTripletArg(c.HostTriplet, "--host")
	if hostTripletArg != "" {
		args = append(args, hostTripletArg)
	}

	targetTripletArg := c.buildTripletArg(c.TargetTriplet, "--target")
	if targetTripletArg != "" {
		args = append(args, targetTripletArg)
	}

	return args
}

func (c *Configure) buildTripletArg(triplet *utils.Triplet, argName string) string {
	if triplet == nil {
		return ""
	}

	return fmt.Sprintf("%s=%s", argName, triplet)
}
