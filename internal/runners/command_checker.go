package runners

import (
	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
)

type CommandChecker struct {
	GenericRunner
	Command string
}

// Note: this depends on the `which` command existing on the system
// TODO consider writing an implementation that does not rely on this command
func (cc CommandChecker) BuildTask() (*execute.ExecTask, error) {
	task, err := cc.GenericRunner.BuildTask()
	if err != nil {
		return task, trace.Wrap(err, "failed to create generic runner task")
	}

	task.Command = "which"
	task.Args = []string{cc.Command}

	return task, nil
}

func (cc CommandChecker) GetCommandPath() (string, error) {
	cmdResult, err := Run(cc)
	if err != nil {
		return "", trace.Wrap(err, "failed to run version checker command")
	}

	if cmdResult.ExitCode == 0 {
		return cmdResult.Stdout, nil
	}

	return "", nil
}

func CheckRequiredCommandsExist(requiredCommands []string) error {
	for _, requiredCommand := range requiredCommands {
		commandPath, err := CommandChecker{
			Command: requiredCommand,
		}.GetCommandPath()
		if err != nil {
			return trace.Wrap(err, "failed to check if command %q exists", requiredCommand)
		}

		if commandPath == "" {
			return trace.Errorf("required command %q does not exist", requiredCommand)
		}
	}
	return nil
}
