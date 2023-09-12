package utils

import (
	"bufio"
	"os"

	"github.com/gravitational/trace"
)

// Efficiently read a file, split upon newline, into a string array
func ReadLines(filePath string) ([]string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, trace.Wrap(err, "failed to get file at %q", filePath)
	}

	if !fileInfo.Mode().IsRegular() {
		return nil, trace.Wrap(err, "file at %q is not a regular file", filePath)
	}

	fileHandle, err := os.Open(filePath)
	defer func() {
		if fileHandle != nil {
			closeErr := fileHandle.Close()
			if err == nil && closeErr != nil {
				err = trace.Wrap(closeErr, "failed to close file %q", filePath)
			}
		}
	}()
	if err != nil {
		return nil, trace.Wrap(err, "failed to open %q for reading", filePath)
	}

	var lines []string
	fileReader := bufio.NewScanner(fileHandle)
	for fileReader.Scan() {
		lines = append(lines, fileReader.Text())
	}

	return lines, nil
}
