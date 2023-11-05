package utils

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gravitational/trace"
)

// Searches the path, using the PATH environement variable for the current process, for a given file name.
// The path to the first match is returned.
// If none are found, an error is returned.
func SearchPath(name string) (string, error) {
	filePath, err := exec.LookPath(name)
	if err == nil {
		return filePath, nil

	}

	if errors.Is(err, exec.ErrDot) {
		absolutePath, err := filepath.Abs(filePath)
		if err != nil {
			return "", trace.Wrap(err, "failed to resolve %q to an absolute path", filePath)
		}

		return absolutePath, nil
	}

	return "", trace.Wrap(err, "failed to look up %q in the PATH search path", name)
}

func DoesFilesystemPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, trace.Wrap(err, "failed to stat filepath %q", path)
}
