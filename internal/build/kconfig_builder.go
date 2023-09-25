package build

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

// TODO consider making this more generic, "ConfigBuilder" or similar
type IKconfigBuilder interface {
	SetConfigFilePath(string)
	GetConfigFilePath() string
}

type KconfigBuilder struct {
	ConfigFilePath string
}

func (kb *KconfigBuilder) SetConfigFilePath(configFilePath string) {
	kb.ConfigFilePath = configFilePath
}

func (kb *KconfigBuilder) GetConfigFilePath() string {
	return kb.ConfigFilePath
}

func (kb *KconfigBuilder) CopyKconfig(buildDirectoryPath string, replacementVars map[string]args.IValue) error {
	lineChannel, errChannel := utils.StreamLines(kb.ConfigFilePath)

	// This could possibly be made slightly safer/more clear if there is an issue
	// by waiting for the first line to become availble before opening the destination
	// file for writing. The tradeoff is that opening the destination file would not
	// occur until after the first line is read, making this code slower.
	destinationFilePath := path.Join(buildDirectoryPath, ".config")
	destinationFile, err := os.OpenFile(destinationFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer utils.Close(destinationFile, &err)
	if err != nil {
		return trace.Wrap(err, "failed to open destination file %q for writing", destinationFilePath)
	}

	// Write the replacement variables along with comment headers for the prefix and source file

	// Write the variables from the source file, less replacement variables
	err = kb.filterCopyFile(lineChannel, errChannel, destinationFile, replacementVars)
	if err != nil {
		return trace.Wrap(err, "failed to copy config file at %q to %q", kb.ConfigFilePath, destinationFilePath)
	}

	return nil
}

func (kb *KconfigBuilder) filterCopyFile(lineChannel <-chan string, errChannel <-chan error, destinationFile *os.File, replacementVars map[string]args.IValue) error {
	for {
		select {
		case err := <-errChannel:
			return trace.Wrap(err, "failed to read kconfig file at %q", kb.ConfigFilePath)
		case line, more := <-lineChannel:
			// Check if the line starts with a variable that is being replaced
			trimmedLine := strings.TrimSpace(line)
			for varName, varValue := range replacementVars {
				if strings.HasPrefix(trimmedLine, fmt.Sprintf("%s=", varName)) {
					line = fmt.Sprintf("%s=%q", varName, varValue.GetValue())
					break
				}
			}

			_, err := destinationFile.WriteString(fmt.Sprintf("%s\n", line))
			if err != nil {
				return trace.Wrap(err, "failed to write line %q to destination file", line)
			}

			if !more {
				return nil
			}
		}
	}
}
