package utils

import (
	"errors"
	"io/fs"
	"os"
	"path"

	"github.com/gravitational/trace"
)

// Link name (new file): link target (old file)
// Symlinks will target the link target exactly, without prepending the output directory path
func CreateSymlinks(links map[string]string, symlinkPrefixPath string) error {
	for symlink, targetPath := range links {
		symlinkPath := path.Join(symlinkPrefixPath, symlink)

		err := CreateSymlink(symlinkPath, targetPath)
		if err != nil {
			return trace.Wrap(err, "failed to create symlink %q targeting %q", symlinkPath, targetPath)
		}
	}

	return nil
}

func CreateSymlink(symlinkPath, targetPath string) error {
	// Check if the file exists already
	newFileInfo, err := os.Stat(symlinkPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return trace.Wrap(err, "failed to retreive file info for new file %q", symlinkPath)
	}

	// If the file info is not null then a filesystem entry exists at this path
	if newFileInfo != nil {
		if newFileInfo.Mode().Type() != fs.ModeSymlink {
			return trace.Errorf("the new file %q already exists and is not a symlink", symlinkPath)
		}

		// Remove the file if exists as it will not be updated by os.Symlink
		err = os.Remove(symlinkPath)
		if err != nil {
			return trace.Wrap(err, "failed to delete symlink at %q", symlinkPath)
		}
	}

	err = os.Symlink(targetPath, symlinkPath)
	if err != nil {
		return trace.Wrap(err, "failed to symlink %q to target %q", symlinkPath, targetPath)
	}

	return nil
}
