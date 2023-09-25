package build

import (
	"os"
	"strings"

	"github.com/gravitational/trace"
	cp "github.com/otiai10/copy"
)

type ISourceBuilder interface {
	SetSourceDirectoryPath(string)
	GetSourceDirectoryPath() string
}

type SourceBuilder struct {
	SourceDirectoryPath string
}

func (sb *SourceBuilder) SetSourceDirectoryPath(sourceDirectoryPath string) {
	// Default value
	if sourceDirectoryPath == "" {
		sb.SourceDirectoryPath = "/tmp/source" // TODO move this to appropriate place under /var
		return
	}

	sb.SourceDirectoryPath = sourceDirectoryPath
}

func (sb *SourceBuilder) GetSourceDirectoryPath() string {
	return sb.SourceDirectoryPath
}

// TODO rework this so that sb.SourceDirectoryPath is updated d uring the build process
func (sb *SourceBuilder) CopyToBuildDirectory(sourceDirectoryPath, buildDirectoryPath string) error {
	err := cp.Copy(sourceDirectoryPath, buildDirectoryPath, cp.Options{
		// Skip copying ".git*" files, such as the ".git" directory and ".gitignore"
		Skip: func(srcInfo os.FileInfo, src, dest string) (bool, error) {
			return strings.HasPrefix(srcInfo.Name(), ".git"), nil
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to copy source directory %q contents to build directory %q", sb.SourceDirectoryPath, buildDirectoryPath)
	}

	return nil
}
