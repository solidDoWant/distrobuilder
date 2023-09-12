package source

import (
	"os"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type Source struct {
	DownloadPath          string // Directory to download the source to, relative to the current working directory
	shouldDeleteOnCleanup bool
}

func NewSource(downloadPath string) *Source {
	if downloadPath == "" {
		downloadPath = utils.GetTempDirectoryPath()
	}

	return &Source{
		DownloadPath:          downloadPath,
		shouldDeleteOnCleanup: false, // False by default, will be set to true if the download path is determined to not exist yet
	}
}

func (s *Source) Setup() error {
	didPathAlreadyExist, err := utils.EnsureDirectoryExists(s.DownloadPath)
	s.shouldDeleteOnCleanup = !didPathAlreadyExist
	if err != nil {
		return trace.Wrap(err, "failed to ensure download directory exists with correct permissions and ownership")
	}

	return nil
}

func (s *Source) Cleanup() error {
	if !s.shouldDeleteOnCleanup {
		return nil
	}

	err := os.RemoveAll(s.DownloadPath)
	if err != nil {
		return trace.Wrap(err, "failed to delete download directory %q", s.DownloadPath)
	}

	return nil
}
