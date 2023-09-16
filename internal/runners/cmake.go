package runners

import (
	"fmt"
	"log/slog"
	"path"
	"strings"

	execute "github.com/alexellis/go-execute/pkg/v1"
	"github.com/gravitational/trace"
	"github.com/pbnjay/memory"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type CMakeDefines map[string]string

func (cmds *CMakeDefines) AsArgs() []string {
	formattedDefines := make(map[string]string, len(*cmds))
	for name, value := range *cmds {
		formattedDefines[fmt.Sprintf("-D%s", name)] = value
	}

	return mapToArgs(formattedDefines)
}

type CMake struct {
	GenericRunner
	Generator string
	Undefines []string
	Defines   CMakeDefines
	Caches    []string
	Path      string
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

	for _, undefine := range cm.Undefines {
		args = append(args, fmt.Sprintf("-U%s", undefine))
	}

	args = append(args, cm.Defines.AsArgs()...)

	for _, cache := range cm.Caches {
		args = append(args, "-C", cache)
	}

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
