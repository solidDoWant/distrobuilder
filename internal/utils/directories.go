package utils

import (
	"io/fs"
	"log/slog"
	"os"
	"os/user"
	"path"
	"strconv"

	"github.com/google/uuid"
	"github.com/gravitational/trace"
)

func EnsureTempDirectoryExists() (string, error) {
	directoryPath := GetTempDirectoryPath()
	didPathAlreadyExist, err := EnsureDirectoryExists(directoryPath)
	if err != nil {
		return directoryPath, trace.Wrap(err, "failed to create temporary directory")
	}

	if didPathAlreadyExist {
		return directoryPath, trace.Errorf("temporary path %q already exists", directoryPath)
	}

	return directoryPath, nil
}

func GetTempDirectoryPath() string {
	return path.Join(os.TempDir(), uuid.New().String())
}

func EnsureDirectoryExists(directoryPath string) (bool, error) {
	var directoryPathStat fs.FileInfo
	var err error
	didDirectoryAlreadyExist := true
	for {
		directoryPathStat, err = os.Stat(directoryPath)
		if err == nil {
			break
		}

		directoryFilemode := os.FileMode(0770) // Read, write, and execute permissions for the current user and group
		if os.IsNotExist(err) {
			slog.Debug("Directory does not exist, creating it", "directory_path", directoryPath)
			didDirectoryAlreadyExist = false

			err = os.MkdirAll(directoryPath, directoryFilemode)
			if err != nil {
				return false, trace.Wrap(err, "failed to create path at %q", directoryPath)
			}

			continue
		}

		// The directory may or may not exist at this point... take a guess for error returns
		if os.IsPermission(err) {
			// If the directoryPath exists but permissions are wrong
			slog.Debug("Current user does not have access to path, updating permissions", "directory_path", directoryPath)
			err = os.Chmod(directoryPath, directoryFilemode)
			if err != nil {
				return false, trace.Wrap(err, "failed to change path %q file mode to %o", directoryPath, directoryFilemode)
			}

			continue
		}

		return false, trace.Wrap(err, "failed to stat %q", directoryPath)
	}

	if !directoryPathStat.IsDir() {
		return false, trace.Errorf("path %q exists but is not a directory", directoryPath)
	}

	err = setDirectoryOwner(directoryPath)
	if err != nil {
		return didDirectoryAlreadyExist, trace.Wrap(err, "failed to ensure path %q has correct user and group ownership", directoryPath)
	}

	return didDirectoryAlreadyExist, nil
}

func setDirectoryOwner(directoryPath string) error {
	currentUser, err := user.Current()
	if err != nil {
		return trace.Errorf("failed to get current user")
	}

	currentUserId, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return trace.Wrap(err, "failed to parse current user ID %q into integer", currentUser.Uid)
	}

	currentUserPrimaryGroupId, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		return trace.Wrap(err, "failed to parse current user primary group ID %q into integer", currentUser.Gid)
	}

	err = os.Chown(directoryPath, currentUserId, currentUserPrimaryGroupId)
	if err != nil {
		currentUserPrimaryGroup, err := user.LookupGroupId(currentUser.Gid)
		if err != nil {
			if _, ok := err.(user.UnknownGroupIdError); !ok {
				return trace.Wrap(err, "failed to lookup group name for group ID %s", currentUser.Gid)
			}
		}
		return trace.Wrap(err, "failed to set path %q directory ownership to %d:%d (%s:%s)", directoryPath, currentUserId, currentUserPrimaryGroupId, currentUser.Name, currentUserPrimaryGroup.Name)
	}

	return nil
}
