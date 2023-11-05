package runners

import (
	"fmt"
	"os"
	"path"

	execute "github.com/alexellis/go-execute/pkg/v1"
	pie "github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type MesonOptions struct {
	Options    map[string]args.IValue
	CrossFile  map[string]map[string]args.IValue // Keyed by section name (i.e `binaries` or `host_machine`), with a map of each section variable (i.e. `c` or `system`) and value (i.e. `clang` or `linux`)
	NativeFile map[string]map[string]args.IValue // Keyed by section name (i.e `binaries` or `host_machine`), with a map of each section variable (i.e. `c` or `system`) and value (i.e. `clang` or `linux`)
}

func MergeMesonOptions(options ...*MesonOptions) (*MesonOptions, error) {
	options = utils.FilterNil(options)

	mergedOptions, err := args.MergeMap(pie.Map(options, func(option *MesonOptions) map[string]args.IValue { return option.Options })...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge all Meson option values")
	}

	mergedCrossFileOptions, err := args.MergeNestedMap(pie.Map(options, func(option *MesonOptions) map[string]map[string]args.IValue { return option.CrossFile })...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge meson cross file options")
	}

	mergedNativeFileOptions, err := args.MergeNestedMap(pie.Map(options, func(option *MesonOptions) map[string]map[string]args.IValue { return option.NativeFile })...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge meson native file options")
	}

	return &MesonOptions{
		Options:    mergedOptions,
		CrossFile:  mergedCrossFileOptions,
		NativeFile: mergedNativeFileOptions,
	}, nil
}

type Meson struct {
	GenericRunner
	Backend             string
	SourceDirectoryPath string
	BuildDirectoryPath  string
	Options             []*MesonOptions
}

func (m *Meson) Setup() error {
	mergedOptions, err := MergeMesonOptions(m.Options...)
	if err != nil {
		return trace.Wrap(err, "failed to merge CMake options")
	}

	// Create the meson cross file
	err = m.generateConfigFile("cross", mergedOptions.CrossFile)
	if err != nil {
		return trace.Wrap(err, "failed to generate cross file")
	}

	// Create the meson native file
	err = m.generateConfigFile("native", mergedOptions.NativeFile)
	if err != nil {
		return trace.Wrap(err, "failed to generate native file")
	}

	return nil
}

func (m *Meson) generateConfigFile(configType string, configData map[string]map[string]args.IValue) error {
	configFilePath := m.getConfigFilePath(configType)
	fileHandle, err := os.Create(configFilePath)
	defer utils.Close(fileHandle, &err)
	if err != nil {
		return trace.Wrap(err, "failed to open meson %s file %q for writing", configType, configFilePath)
	}

	// Produce a file following the format at https://mesonbuild.com/Cross-compilation.html, which is the same for native and cross files
	for section, data := range configData {
		if len(data) == 0 {
			continue
		}

		_, err = fileHandle.WriteString(fmt.Sprintf("[%s]\n", section))
		if err != nil {
			return trace.Wrap(err, "failed to write section header %q to meson %s file at %q", section, configType, configFilePath)
		}

		for key, value := range data {
			lineEntry := fmt.Sprintf("%s = '%s'\n", key, value.GetValue())
			_, err = fileHandle.WriteString(lineEntry)
			if err != nil {
				return trace.Wrap(err, "failed to write line entry %q for section %q to meson %s file at %q", lineEntry, section, configType, configFilePath)
			}
		}
	}

	return nil
}

func (m *Meson) BuildTask() (*execute.ExecTask, error) {
	task, err := m.GenericRunner.BuildTask()
	if err != nil {
		return task, trace.Wrap(err, "failed to create generic runner task")
	}

	args, err := m.buildArgs()
	if err != nil {
		return nil, trace.Wrap(err, "failed to create runner args")
	}

	task.Args = append(task.Args, args...)
	task.Command = "meson"

	return task, nil
}

func (m *Meson) buildArgs() ([]string, error) {
	var commandArgs []string

	commandArgs = append(commandArgs, "setup")

	mergedOptions, err := MergeMesonOptions(m.Options...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge CMake options")
	}

	mergedOptions.Options["backend"] = args.StringValue("ninja")

	commandArgs = append(commandArgs, "--reconfigure", "--clearcache")
	commandArgs = append(commandArgs, fmt.Sprintf("--cross-file=%s", m.getConfigFilePath("cross")))
	commandArgs = append(commandArgs, fmt.Sprintf("--native-file=%s", m.getConfigFilePath("native")))

	commandArgs = append(commandArgs,
		pie.Map(pie.Keys(mergedOptions.Options), func(varName string) string {
			return fmt.Sprintf("-D%s=%s", varName, mergedOptions.Options[varName].GetValue())
		})...,
	)

	return append(commandArgs, m.BuildDirectoryPath, m.SourceDirectoryPath), nil
}

func (m *Meson) getConfigFilePath(configType string) string {
	return path.Join(m.BuildDirectoryPath, fmt.Sprintf("meson-%s-file.txt", configType))
}
