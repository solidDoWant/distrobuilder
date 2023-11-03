package utils

import (
	"bufio"
	"io"
	"net/http"
	"os"

	"github.com/gravitational/trace"
)

func DownloadFile(url, destinationFilePath string) error {
	fileHandle, err := os.Create(destinationFilePath)
	defer Close(fileHandle, &err)
	if err != nil {
		return trace.Wrap(err, "failed to open download file path %q for writing", fileHandle)
	}

	httpResponse, err := http.Get(url)
	defer Close(httpResponse.Body, &err)
	if err != nil {
		return trace.Wrap(err, "failed to get HTTP response for download URL %q", url)
	}

	writtenByteCount, err := io.Copy(fileHandle, httpResponse.Body)
	if err != nil {
		return trace.Wrap(err, "failed to write HTTP response body to download file path %q", destinationFilePath)
	}

	if httpResponse.ContentLength != -1 && writtenByteCount != httpResponse.ContentLength {
		return trace.Errorf("Only wrote %d bytes to %q, but %d bytes were reported in the content length", writtenByteCount, destinationFilePath, httpResponse.ContentLength)
	}

	return nil
}

func ReadLines(filePath string) ([]string, error) {
	lineChannel, errChannel := StreamLines(filePath)

	var lines []string
	for {
		select {
		case err := <-errChannel:
			return lines, err
		case line, more := <-lineChannel:
			lines = append(lines, line)
			if !more {
				return lines, nil
			}
		}
	}
}

// Efficiently read a file, split upon newline, sending each line out via the channel
func StreamLines(filePath string) (<-chan string, <-chan error) {
	outChannel := make(chan string, 10) // The size here is arbitrary. Any value greater than 1 should decrease context switches at thet cost of memory.
	errChannel := make(chan error)

	go func() {
		defer close(outChannel)
		defer close(errChannel)

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			errChannel <- trace.Wrap(err, "failed to get file at %q", filePath)
			return
		}

		if !fileInfo.Mode().IsRegular() {
			errChannel <- trace.Wrap(err, "file at %q is not a regular file", filePath)
			return
		}

		fileHandle, err := os.Open(filePath)
		defer CloseChannel(fileHandle, errChannel)
		if err != nil {
			errChannel <- trace.Wrap(err, "failed to open %q for reading", filePath)
			return
		}

		fileReader := bufio.NewScanner(fileHandle)

		for fileReader.Scan() {
			err = fileReader.Err()
			if err != nil {
				errChannel <- trace.Wrap(err, "an error occured while reading %q", filePath)
				return
			}
			outChannel <- fileReader.Text()
		}
	}()

	return outChannel, errChannel
}
