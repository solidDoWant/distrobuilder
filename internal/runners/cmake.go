package runners

import (
	"fmt"
	"log/slog"
	"path"
	"strings"

	execute "github.com/alexellis/go-execute/pkg/v1"
	pie "github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/pbnjay/memory"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type CMakeOptions struct {
	Undefines []string
	Defines   map[string]args.IValue
	Caches    []string
}

func CommonOptions() *CMakeOptions {
	return &CMakeOptions{
		Defines: map[string]args.IValue{
			"CMAKE_BUILD_TYPE":  args.StringValue("Release"),
			"CMAKE_SYSTEM_NAME": args.StringValue("Linux"), // This will enable cross compiling when set
		},
	}
}

func DebugOptions() *CMakeOptions {
	toolFlags := args.SeparatorValues(" ", "-v") // "-v" will tell clang to output the full commands that it generates along with some other info
	return &CMakeOptions{
		Defines: map[string]args.IValue{
			"CMAKE_C_FLAGS":             toolFlags,
			"CMAKE_CXX_FLAGS":           toolFlags,
			"CMAKE_EXE_LINKER_FLAGS":    toolFlags,
			"CMAKE_MODULE_LINKER_FLAGS": toolFlags,
			"CMAKE_SHARED_LINKER_FLAGS": toolFlags,
			"CMAKE_STATIC_LINKER_FLAGS": toolFlags,
		},
	}
}

func MergeCMakeOptions(options ...*CMakeOptions) (*CMakeOptions, error) {
	options = utils.FilterNil(options)
	mergedDefines, err := args.MergeMap(pie.Map(options, func(option *CMakeOptions) map[string]args.IValue { return option.Defines })...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge all CMake variable define values")
	}

	return &CMakeOptions{
		Undefines: utils.DedupeReduce(pie.Map(options, func(option *CMakeOptions) []string { return option.Undefines })...),
		Defines:   mergedDefines,
		Caches:    utils.DedupeReduce(pie.Map(options, func(option *CMakeOptions) []string { return option.Caches })...),
	}, nil
}

type CMake struct {
	GenericRunner
	Generator string
	Path      string
	Options   []*CMakeOptions
}

func (cm CMake) BuildTask() (*execute.ExecTask, error) {
	task, err := cm.GenericRunner.BuildTask()
	if err != nil {
		return task, trace.Wrap(err, "failed to create generic runner task")
	}

	args, err := cm.buildArgs()
	if err != nil {
		return nil, trace.Wrap(err, "failed to create runner args")
	}

	task.Args = append(task.Args, args...)
	task.Command = "cmake"

	return task, nil
}

func (cm *CMake) buildArgs() ([]string, error) {
	var args []string

	if cm.Generator != "" {
		args = append(args, "-G", cm.Generator)
	}

	mergedOptions, err := MergeCMakeOptions(cm.Options...)
	if err != nil {
		return nil, trace.Wrap(err, "failed to merge CMake options")
	}

	args = pie.Flat([][]string{
		args,
		pie.Of(mergedOptions.Undefines).Map(func(s string) string { return fmt.Sprintf("-U%s", s) }).Result,
		pie.Map(pie.Keys(mergedOptions.Defines), func(varName string) string {
			return fmt.Sprintf("-D%s=%s", varName, mergedOptions.Defines[varName].GetValue())
		}),
		pie.Of(mergedOptions.Caches).Map(func(s string) string { return fmt.Sprintf("-C %s", s) }).Result,
	})

	if cm.Path == "" {
		cm.Path = "."
	}

	return append(args, cm.Path), nil
}

func GetCmakeMaxRecommendedParallelLinkJobs() int {
	// Typical recommendations are 1 job for every available 15 GB of RAM to keep the process from being OOM killed
	freeMemoryBytes := memory.FreeMemory()
	var recommendedBytesPerJob uint64 = 15 * 1024 * 1024 * 1024 // 15 GB

	recommendedJobCount := freeMemoryBytes / recommendedBytesPerJob
	if recommendedJobCount == 0 {
		recommendedJobCount = 1 // Default to one job or nothing can be done
	}

	return int(recommendedJobCount)
}

func GetCmakeCacheVars(buildDirectoryPath string) (map[string]string, error) {
	cacheFilePath := path.Join(buildDirectoryPath, "CMakeCache.txt")

	cacheLines, err := utils.ReadLines(cacheFilePath)
	if err != nil {
		return nil, trace.Wrap(err, "failed to read file lines from CMake cache at %q", cacheFilePath)
	}

	cacheVars := make(map[string]string, len(cacheLines))
	for _, cacheLine := range cacheLines {
		trimmedLine := strings.TrimSpace(cacheLine)

		// Skip empty lines
		if trimmedLine == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Extract the key and value
		keySection, value, isValidLine := strings.Cut(trimmedLine, "=")
		if !isValidLine {
			return cacheVars, trace.Wrap(err, "found non-cache var line in CMake cache file %q: %q", cacheFilePath, cacheLine)
		}

		// Extract the key, trimming extra data like the type
		key := keySection
		lastColonIndex := strings.LastIndex(keySection, ":")
		if lastColonIndex != -1 {
			key = keySection[:lastColonIndex]
		}

		cacheVars[key] = value
	}

	slog.Debug("loaded cmake cache vars", "vars", cacheVars)

	return cacheVars, nil
}
