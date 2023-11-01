package build

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/runners"
	"github.com/solidDoWant/distrobuilder/internal/runners/args"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type LibreSSL struct {
	StandardBuilder
}

func NewLibreSSL() *LibreSSL {
	instance := &LibreSSL{
		StandardBuilder: StandardBuilder{
			Name: "libressl",
			BinariesToCheck: []string{
				path.Join("usr", "bin", "openssl"),
				path.Join("usr", "lib", "libcrypto.so"),
				path.Join("usr", "lib", "libssl.so"),
				path.Join("usr", "lib", "libtls.so"),
			},
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (lssl *LibreSSL) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewLibreSSLGitRepo(repoDirectoryPath, ref)
}

func (lssl *LibreSSL) DoConfiguration(buildDirectoryPath string) error {
	err := lssl.Autogen(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to run autogen for %s build", lssl.Name)
	}

	cmakeBuildDirectory := lssl.getCmakeBuildDirectory(buildDirectoryPath)
	_, err = utils.EnsureDirectoryExists(cmakeBuildDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to create CMake build directory at %q", cmakeBuildDirectory)
	}

	// TODO add support for building static libs as well
	cmakeOptions := &runners.CMakeOptions{
		Defines: map[string]args.IValue{
			"ENABLE_NC":         args.OnValue(),
			"BUILD_SHARED_LIBS": args.OnValue(),
		},
	}
	// Source and build directory are the same as it's being built in tree, copied to the build
	// directory with autogen ran
	return lssl.CMakeConfigureWithPath(cmakeBuildDirectory, buildDirectoryPath, cmakeOptions)
}

func (lssl *LibreSSL) DoBuild(buildDirectoryPath string) error {
	cmakeBuildDirectory := lssl.getCmakeBuildDirectory(buildDirectoryPath)
	err := lssl.NinjaBuild(cmakeBuildDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to build %s with Ninja", lssl.Name)
	}

	err = lssl.moveEtcDirectory()
	if err != nil {
		return trace.Wrap(err, "failed to move built etc directory")
	}

	err = lssl.patchBinaryEtcReferences()
	if err != nil {
		return trace.Wrap(err, "failed to patch all binary etc references")
	}

	return nil
}

func (lssl *LibreSSL) moveEtcDirectory() error {
	etcSourceDirectory := lssl.getInstallPath(path.Join("usr", "etc"))
	etcDestDirectory := lssl.getInstallPath("etc")
	err := os.Rename(etcSourceDirectory, etcDestDirectory)
	if err != nil {
		return trace.Wrap(err, "failed to move etc directory from %q to %q", etcSourceDirectory, etcDestDirectory)
	}

	return nil
}

// libressl will reverence files in <output directory>/usr/etc unless it is built
// without the install path set, which would modify the host build system. This
// function patches any binaries (executables, libraries) with the output directory
// listed.
func (lssl *LibreSSL) patchBinaryEtcReferences() error {
	executablesToPatch := []string{
		"openssl",
		"nc",
		"ocspcheck",
	}

	librariesToCheck := []string{
		"libtls.so",
		"libcrypto.so",
	}

	searchPrefix := path.Join(lssl.OutputDirectoryPath, "usr")

	filesToPatch := make([]string, 0, len(executablesToPatch)+len(librariesToCheck))
	for _, executableToPatch := range executablesToPatch {
		filesToPatch = append(filesToPatch, path.Join("bin", executableToPatch))
	}
	for _, libraryToCheck := range librariesToCheck {
		filesToPatch = append(filesToPatch, path.Join("lib", libraryToCheck))
	}

	for _, fileToPatch := range filesToPatch {
		filePath := path.Join(lssl.OutputDirectoryPath, "usr", fileToPatch)

		err := lssl.patchBinaryEtcReference(filePath, searchPrefix)
		if err != nil {
			return trace.Wrap(err, "failed to patch binary at %q", filePath)
		}
	}

	return nil
}

func (lssl *LibreSSL) patchBinaryEtcReference(filePath, searchPrefix string) error {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return trace.Wrap(err, "failed to read file %q into memory", filePath)
	}

	// TODO look in data sections only?

	fileStrings := findNullTerminatedStrings(fileBytes, len(searchPrefix))
	for fileString, positions := range fileStrings {
		startOffset := strings.Index(fileString, searchPrefix)

		// Skip the string if the search prefix is not found in it
		if startOffset == -1 {
			continue
		}

		// Remove the search prefix from the portion of the file string that contains it
		correctPath, err := filepath.Rel(searchPrefix, fileString[startOffset:])
		if err != nil {
			return trace.Wrap(err, "failed to get path of %q relative to %q", fileString, searchPrefix)
		}

		// Make the path absolute, and add a null termination character
		replacementValue := []byte("/" + correctPath + string(rune(0)))

		// Copy the replacement value
		for _, position := range positions {
			utils.UpdateSubset(&fileBytes, startOffset+position, &replacementValue)
		}
	}

	err = os.WriteFile(filePath, fileBytes, 0)
	if err != nil {
		return trace.Wrap(err, "failed to write patched binary to %q", filePath)
	}

	return nil
}

func findNullTerminatedStrings(bytes []byte, minLength int) map[string][]int {
	results := make(map[string][]int)
	minLength++ // Increase by 1 to account for the null termination character

	previousNullTerminationCharacterPosition := 0
	for i, b := range bytes {
		// Find null termination characters
		if rune(b) != rune(0) {
			continue
		}

		// Found a string of approprate length
		if i-previousNullTerminationCharacterPosition >= minLength {
			// Skip the last null termination character, and the current one
			startPosition := previousNullTerminationCharacterPosition + 1
			foundString := string(bytes[startPosition:i])
			results[foundString] = append(results[foundString], startPosition)
		}

		previousNullTerminationCharacterPosition = i
	}

	return results
}

func (lssl *LibreSSL) getCmakeBuildDirectory(buildDirectoryPath string) string {
	return path.Join(buildDirectoryPath, "build")
}
