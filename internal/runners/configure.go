package runners

import (
	"fmt"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type Configure struct {
	GenericRunner
	CCompiler           string
	CFlags              string
	CppCompiler         string
	InstallPath         string
	SourceDirectoryPath string
	ConfigurePath       string
	HostTriplet         *utils.Triplet
	TargetTriplet       *utils.Triplet
	AdditionalFlags     map[string]string // These values will be converted to a "key=value" output, skipping if "key" is empty
	AdditionalArgs      []string          // These values will be appended exactly as is
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

	// Temp debugging
	// task.Command = "/workspaces/distrobuilder/test.sh"
	// task.Args = append([]string{c.ConfigurePath}, task.Args...)

	task.Args = append(task.Args, args...)
	task.Command = c.ConfigurePath

	return task, nil
}

func (c *Configure) buildArgs() ([]string, error) {
	args := []string{}

	if c.InstallPath != "" {
		args = append(args, fmt.Sprintf("--prefix=%s", c.InstallPath))
	}

	if c.SourceDirectoryPath != "" {
		args = append(args, fmt.Sprintf("--srcdir=%s", c.SourceDirectoryPath))
	}

	tripletArgs, err := c.buildTripletArgs()
	if err != nil {
		return nil, trace.Wrap(err, "failed to create triplet args")
	}
	args = append(args, tripletArgs...)

	args = append(args, mapToArgs(c.AdditionalFlags)...)
	args = append(args, c.AdditionalArgs...)

	args = append(args, c.buildVariableArgs()...)

	return args, nil
}

func (c *Configure) buildVariableArgs() []string {
	args := []string{}

	if c.CCompiler != "" {
		args = append(args, fmt.Sprintf("CC=%s", c.CCompiler))
	}

	if c.CFlags != "" {
		args = append(args, fmt.Sprintf("CFLAGS=%s", c.CFlags))
	}

	if c.CppCompiler != "" {
		args = append(args, fmt.Sprintf("CXX=%s", c.CppCompiler))
	}

	return args
}

func (c *Configure) buildTripletArgs() ([]string, error) {
	args := []string{}

	hostTripletArg, err := c.buildTripletArg(c.HostTriplet, "--host")
	if err != nil {
		return nil, trace.Wrap(err, "failed to create host triplet args")
	}
	if hostTripletArg != "" {
		args = append(args, hostTripletArg)
	}

	targetTripletArg, err := c.buildTripletArg(c.TargetTriplet, "--target")
	if err != nil {
		return nil, trace.Wrap(err, "failed to create target triplet args")
	}
	if targetTripletArg != "" {
		args = append(args, targetTripletArg)
	}

	return args, nil
}

func (c *Configure) buildTripletArg(triplet *utils.Triplet, argName string) (string, error) {
	if triplet == nil {
		return "", nil
	}

	targetTriplet, err := triplet.AsString()
	if err != nil {
		return "", trace.Wrap(err, "failed to convert triplet %v to string", triplet)
	}
	return fmt.Sprintf("%s=%s", argName, targetTriplet), nil
}
