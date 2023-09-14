package source

import (
	"os"
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type Source struct {
	DownloadRootDir       string
	DownloadPath          string // Directory to download the source to, relative to the current working directory or download root directory if set
	ShouldDeleteOnCleanup bool
}

func NewSource(downloadRootDir string) *Source {
	shouldDeleteOnCleanup := false
	if downloadRootDir == "" {
		downloadRootDir = utils.GetTempDirectoryPath()
		shouldDeleteOnCleanup = true // Cleanup when a temp directory is created
	}

	return &Source{
		DownloadRootDir:       downloadRootDir,
		ShouldDeleteOnCleanup: shouldDeleteOnCleanup,
	}
}

func (s *Source) Setup() error {
	_, err := utils.EnsureDirectoryExists(s.FullDownloadPath())
	if err != nil {
		return trace.Wrap(err, "failed to ensure download directory exists with correct permissions and ownership")
	}

	return nil
}

func (s *Source) Cleanup() error {
	if !s.ShouldDeleteOnCleanup {
		return nil
	}

	fullDownloadPath := s.FullDownloadPath()
	err := os.RemoveAll(fullDownloadPath)
	if err != nil {
		return trace.Wrap(err, "failed to delete download directory %q", fullDownloadPath)
	}

	return nil
}

func (s *Source) FullDownloadPath() string {
	return path.Join(s.DownloadRootDir, s.DownloadPath)
}
