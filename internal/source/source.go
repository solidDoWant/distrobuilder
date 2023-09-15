package source

import (
	"os"
	"path"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type Source struct {
	DownloadRootDir string
	DownloadPath    string // Directory to download the source to, relative to the current working directory or download root directory if set
}

func NewSource() *Source {
	return &Source{
		DownloadRootDir: path.Join(os.TempDir(), "source"),
	}
}

func (s *Source) Setup() error {
	_, err := utils.EnsureDirectoryExists(s.FullDownloadPath())
	if err != nil {
		return trace.Wrap(err, "failed to ensure download directory exists with correct permissions and ownership")
	}

	return nil
}

func (s *Source) FullDownloadPath() string {
	return path.Join(s.DownloadRootDir, s.DownloadPath)
}

func GetDefaultSourceParentDirectoryPath() string {
	return path.Join(utils.GetTempDirectoryPath(), "source")
}
