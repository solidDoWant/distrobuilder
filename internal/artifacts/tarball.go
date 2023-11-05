package artifacts

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"strings"
	"syscall"

	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
	"golang.org/x/sys/unix"
)

type Tarball struct {
	OutputPath       string
	SourcePath       string
	ShouldResetOwner bool // True to change owner/group to root/root for packaging, and to set owner/group to the current user/primary group for installing
}

func (t Tarball) Install(ctx context.Context, options *InstallOptions) error {
	err := t.validateInstallOptions(options)
	if err != nil {
		return trace.Wrap(err, "failed to validate install options")
	}

	_, err = utils.EnsureDirectoryExists(options.InstallPath)
	if err != nil {
		return trace.Wrap(err, "failed to ensure that install path %q exists", options.InstallPath)
	}

	err = t.extractFilesFromTarball(options.SourcePath, options.InstallPath)
	if err != nil {
		return trace.Wrap(err, "failed to extract files from tarball %q to destination base path %q", options.SourcePath, options.InstallPath)
	}

	return nil
}

func (t *Tarball) validateInstallOptions(options *InstallOptions) error {
	if options.SourcePath == "" {
		return trace.Errorf("no source path was provided")
	}

	sourceFileInfo, err := os.Lstat(options.SourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return trace.Wrap(err, "source path %q does not exist", options.SourcePath)
		}

		return trace.Wrap(err, "failed to get file info for source path %q", options.SourcePath)
	}

	if !sourceFileInfo.Mode().IsRegular() {
		return trace.Errorf("source path %q is not a regular file", options.SourcePath)
	}

	if options.InstallPath == "" {
		options.InstallPath = "/"
	}

	return nil
}

func (t *Tarball) extractFilesFromTarball(tarballPath, destinationBasePath string) error {
	fileHandle, err := os.Open(tarballPath)
	defer utils.Close(fileHandle, &err)
	if err != nil {
		return trace.Wrap(err, "failed to open source path %q for reading", tarballPath)
	}

	gzipReader, err := gzip.NewReader(fileHandle)
	defer utils.Close(gzipReader, &err)
	if err != nil {
		return trace.Wrap(err, "failed to create a new gzip reader for %q", tarballPath)
	}

	tarReader := tar.NewReader(gzipReader)
	err = t.extractFilesFromArchive(tarReader, destinationBasePath)
	if err != nil {
		return trace.Wrap(err, "failed to extract archive to %q", destinationBasePath)
	}

	return nil
}

func (t *Tarball) extractFilesFromArchive(tarReader *tar.Reader, destinationBasePath string) error {
	for {
		header, err := tarReader.Next()
		if err != nil {
			// When all entries have been read
			if err == io.EOF {
				return nil
			}

			return trace.Wrap(err, "encountered error while reading from source file")
		}

		if header == nil {
			return trace.Errorf("encountered empty tar header while extracting source file")
		}

		err = t.extractFilesystemObject(header, tarReader, destinationBasePath)
		if err != nil {
			return trace.Wrap(err, "failed to extract filesystem object to %q", destinationBasePath)
		}
	}
}

func (t *Tarball) extractFilesystemObject(header *tar.Header, tarReader *tar.Reader, destinationBasePath string) error {
	switch header.Typeflag {
	case tar.TypeReg:
		err := t.extractFile(header, tarReader, destinationBasePath)
		if err != nil {
			return trace.Wrap(err, "failed to extract file %q", header.Name)
		}

	case tar.TypeSymlink:
		err := t.extractSymlink(header, destinationBasePath)
		if err != nil {
			return trace.Wrap(err, "failed to extract symlink %q", header.Name)
		}

	case tar.TypeChar:
		err := t.extractCharacterFile(header, destinationBasePath)
		if err != nil {
			return trace.Wrap(err, "failed to extract character file %q", header.Name)
		}

	case tar.TypeDir:
		err := t.extractDirectory(header, destinationBasePath)
		if err != nil {
			return trace.Wrap(err, "failed to extract directory %q", header.Name)
		}

	default:
		return trace.Errorf("unsupported tar entry type %q", header.Typeflag)
	}

	return nil
}

func (t *Tarball) extractFile(header *tar.Header, tarReader *tar.Reader, destinationBasePath string) error {
	outputFilePath := path.Join(destinationBasePath, header.Name)
	fileInfo := header.FileInfo()
	fileMode := fileInfo.Mode()

	outputFileHandle, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_WRONLY, fileMode)
	defer utils.Close(outputFileHandle, &err)
	if err != nil {
		return trace.Wrap(err, "failed to open %q for writing", outputFilePath)
	}

	copyByteCount, err := io.Copy(outputFileHandle, tarReader)
	if err != nil {
		return trace.Wrap(err, "an error occured while extracting %q to %q", header.Name, outputFilePath)
	}

	fileSize := fileInfo.Size()
	if copyByteCount != fileSize {
		return trace.Errorf("failed to extract %q to %q, expected %d bytes, wrote %d", header.Name, outputFilePath, fileSize, copyByteCount)
	}

	err = t.updateOwnerAndPerms(header, outputFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to update ownership and permissions of %q", outputFilePath)
	}

	return nil
}

func (t *Tarball) extractSymlink(header *tar.Header, destinationBasePath string) error {
	outputFilePath := path.Join(destinationBasePath, header.Name)

	err := os.Symlink(header.Linkname, outputFilePath)
	if err == nil {
		return nil
	}

	if !isSymlinkExistError(err) {
		return trace.Wrap(err, "failed to create symlink %q to %q", outputFilePath, header.Linkname)
	}

	// Attempt to remove the file and recreate
	// This is not attmepted on every file to reduce the number of stat syscalls
	err = os.Remove(outputFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to remove pre-existing symlink at %q", outputFilePath)
	}

	err = os.Symlink(header.Linkname, outputFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to create symlink %q to %q after removing pre-existing file at %q[1]", outputFilePath, header.Linkname)
	}

	return nil
}

func (t *Tarball) extractCharacterFile(header *tar.Header, destinationBasePath string) error {
	outputFilePath := path.Join(destinationBasePath, header.Name)

	doesExist, err := utils.DoesFilesystemPathExist(outputFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to check if character file exists at %q", outputFilePath)
	}

	if doesExist {
		err = os.Remove(outputFilePath)
		if err != nil {
			return trace.Wrap(err, "failed to remove character file at %q", outputFilePath)
		}
	}

	// These reductions in var sizes are not great, but there's nothing I can do about them
	devNumber := unix.Mkdev(uint32(header.Devmajor), uint32(header.Devminor))
	err = syscall.Mknod(outputFilePath, uint32(syscall.S_IFCHR|header.FileInfo().Mode()&fs.ModePerm), int(devNumber))
	if err != nil {
		return trace.Wrap(err, "failed to create device node at %q for device number %d:%d", outputFilePath, header.Devmajor, header.Devminor)
	}

	err = t.updateOwnerAndPerms(header, outputFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to update ownership and permissions of %q", outputFilePath)
	}

	return nil
}

func (t *Tarball) extractDirectory(header *tar.Header, destinationBasePath string) error {
	outputFilePath := path.Join(destinationBasePath, header.Name)
	fileMode := header.FileInfo().Mode()

	err := os.MkdirAll(outputFilePath, fileMode)
	if err != nil {
		return trace.Wrap(err, "failed to create %q", outputFilePath)
	}

	err = t.updateOwnerAndPerms(header, outputFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to update ownership and permissions of %q", outputFilePath)
	}

	return nil
}

func isSymlinkExistError(err error) bool {
	linkErr, ok := err.(*os.LinkError)
	if !ok {
		return false
	}

	syscallErrNumber, ok := linkErr.Err.(syscall.Errno)
	if !ok {
		return false
	}

	return errors.Is(syscallErrNumber, syscall.EEXIST)
}

func (t *Tarball) updateOwnerAndPerms(header *tar.Header, outputFilePath string) error {
	fileMode := header.FileInfo().Mode()
	err := os.Chmod(outputFilePath, fileMode)
	if err != nil {
		return trace.Wrap(err, "failed to update file permissions on %q to %o", outputFilePath, fileMode)
	}

	if !t.ShouldResetOwner {
		err = os.Chown(outputFilePath, header.Uid, header.Gid)
		if err != nil {
			return trace.Wrap(err, "failed to set ownership of %q to %d:%d", outputFilePath, header.Uid, header.Gid)
		}
	}

	return nil
}

func (t *Tarball) Package(ctx context.Context) (string, error) {
	if t.OutputPath == "" {
		t.OutputPath = path.Join(os.TempDir(), fmt.Sprintf("build-%s.tar.gz", uuid.New()))
	}

	_, err := utils.EnsureDirectoryExists(path.Dir(t.OutputPath))
	if err != nil {
		return "", trace.Wrap(err, "failed to ensure package ouutput directory exists")
	}

	fileHandle, err := os.OpenFile(t.OutputPath, os.O_CREATE|os.O_WRONLY, 0644)
	defer utils.Close(fileHandle, &err)
	if err != nil {
		return "", trace.Wrap(err, "failed to create build tarball")
	}

	gzipWriter := gzip.NewWriter(fileHandle)
	defer utils.Close(gzipWriter, &err)

	tarWriter := tar.NewWriter(gzipWriter)
	defer utils.Close(tarWriter, &err)

	err = t.addFilesToArchive(tarWriter)
	if err != nil {
		return "", trace.Wrap(err, "failed to add files to build tarball")
	}

	slog.Info("Packaging complete!", "output_file", fileHandle.Name())
	return fileHandle.Name(), nil
}

func (t *Tarball) addFilesToArchive(archiveWriter *tar.Writer) error {
	err := filepath.Walk(t.SourcePath, func(path string, filesystemObjectInfo os.FileInfo, err error) error {
		if err != nil {
			return trace.Wrap(err, "failed to walk dir %q", path)
		}

		// Skip the root path
		if path == t.SourcePath {
			return nil
		}

		slog.Debug("adding file to archive", "file_path", path)
		filesystemObjectHeader, err := t.getTarHeaderForFSObject(path, filesystemObjectInfo)

		// Write the header to the archive
		err = archiveWriter.WriteHeader(filesystemObjectHeader)
		if err != nil {
			return trace.Wrap(err, "failed to write tar header for %q", path)
		}

		// Copy the file cotnent to the archive
		err = t.copyFileToArchive(path, filesystemObjectInfo, archiveWriter)
		if err != nil {
			return trace.Wrap(err, "failed to copy %q to archive", path)
		}

		return nil
	})

	if err != nil {
		return trace.Wrap(err, "failed to walk over and archive all build files")
	}

	return nil
}

func (t *Tarball) copyFileToArchive(path string, filesystemObjectInfo os.FileInfo, archiveWriter *tar.Writer) error {
	// Regular files don't need to be copied
	if !filesystemObjectInfo.Mode().IsRegular() {
		return nil
	}

	fileHandle, err := os.Open(path)
	defer func() {
		if fileHandle != nil {
			closeErr := fileHandle.Close()
			if closeErr == nil || err != nil {
				return
			}
			err = trace.Wrap(err, "failed to close file handle")
		}
	}()
	if err != nil {
		return trace.Wrap(err, "failed to open %q", path)
	}

	_, err = io.Copy(archiveWriter, fileHandle)
	if err != nil {
		return trace.Wrap(err, "failed to copy file %q to archive", path)
	}

	return nil
}

func (t *Tarball) getTarHeaderForFSObject(objectPath string, filesystemObjectInfo os.FileInfo) (*tar.Header, error) {
	linkTarget, err := t.getTarHeaderLinkTarget(objectPath, filesystemObjectInfo)
	if err != nil {
		return nil, trace.Wrap(err, "failure to get link target for %q", objectPath)
	}

	filesystemObjectHeader, err := tar.FileInfoHeader(filesystemObjectInfo, linkTarget)
	if err != nil {
		return nil, trace.Wrap(err, "failed to create tar header for %q", t.SourcePath)
	}

	relativePath := strings.TrimPrefix(objectPath, t.SourcePath)
	relativePath = strings.TrimPrefix(relativePath, "/")

	filesystemObjectHeader.Name = relativePath

	if t.ShouldResetOwner {
		filesystemObjectHeader.Uid = 0
		filesystemObjectHeader.Gid = 0
		filesystemObjectHeader.Uname = "root"
		filesystemObjectHeader.Gname = "root"
	}

	return filesystemObjectHeader, nil
}

func (t *Tarball) getTarHeaderLinkTarget(path string, filesystemObjectInfo os.FileInfo) (string, error) {
	// If not a symlink, return an empty string
	if filesystemObjectInfo.Mode()&os.ModeSymlink == 0 {
		return "", nil
	}

	// Get the target listed for the symbolic link, but do not resolve the target
	linkTarget, err := os.Readlink(path)
	if err != nil {
		return "", trace.Wrap(err, "failed to read link %q", path)
	}

	return linkTarget, nil
}

func (t *Tarball) SetOutputFilePath(outputFilePath string) {
	t.OutputPath = outputFilePath
}
