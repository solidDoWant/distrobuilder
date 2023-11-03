package build

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/gravitational/trace"
	"github.com/otiai10/copy"
	"github.com/solidDoWant/distrobuilder/internal/source"
	git_source "github.com/solidDoWant/distrobuilder/internal/source/git"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type DejaVuFonts struct {
	StandardBuilder
}

func NewDejaVuFonts() *DejaVuFonts {
	instance := &DejaVuFonts{
		StandardBuilder: StandardBuilder{
			Name: "DejaVuFonts",
			// BinariesToCheck: []string{
			// path.Join("usr", "bin", "DejaVuFonts"),
			// path.Join("usr", "bin", "unDejaVuFonts"),
			// path.Join("usr", "lib", "libDejaVuFonts.so"),
			// },
		},
	}

	instance.IStandardBuilder = instance
	return instance
}

func (z *DejaVuFonts) GetGitRepo(repoDirectoryPath string, ref string) *source.GitRepo {
	return git_source.NewDejaVuFontsGitRepo(repoDirectoryPath, ref)
}

func (z *DejaVuFonts) DoConfiguration(buildDirectoryPath string) error {
	err := z.CopyToBuildDirectory(buildDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to copy sources to build directory")
	}

	resourceDirectoryPath := path.Join(buildDirectoryPath, "resources")

	blocksFilePath := path.Join(resourceDirectoryPath, "Blocks.txt")
	err = utils.DownloadFile("https://www.unicode.org/Public/UCD/latest/ucd/Blocks.txt", blocksFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to download Unicode blocks file")
	}

	unicodeDataFilePath := path.Join(resourceDirectoryPath, "UnicodeData.txt")
	err = utils.DownloadFile("https://www.unicode.org/Public/UCD/latest/ucd/UnicodeData.txt", unicodeDataFilePath)
	if err != nil {
		return trace.Wrap(err, "failed to download Unicode data file")
	}

	repo := git_source.NewFontConfigGitRepo("", "")
	err = repo.Download(context.Background())
	if err != nil {
		return trace.Wrap(err, "failed to download FontConfig source")
	}
	err = copy.Copy(path.Join(repo.FullDownloadPath(), "fc-lang"), path.Join(resourceDirectoryPath, "fc-lang"))
	if err != nil {
		return trace.Wrap(err, "failed to copy fc-lang to the %q directory", resourceDirectoryPath)
	}

	return nil
}

func (z *DejaVuFonts) DoBuild(buildDirectoryPath string) error {
	z.MakeBuild(buildDirectoryPath, nil, "all")

	// Copy the files to the output directory with the appropriate file structure
	buildOutputDirectoryPath := path.Join(buildDirectoryPath, "build")
	shareOutputFilesystemPath := path.Join(z.OutputDirectoryPath, "usr", "share")

	// Copy the fonts
	fontsDirectoryPath := path.Join(shareOutputFilesystemPath, "fonts")
	err := copy.Copy(buildOutputDirectoryPath, fontsDirectoryPath, copy.Options{
		PreserveTimes:     true,
		PermissionControl: copy.AddPermission(0644),
		Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
			return path.Ext(src) != ".ttf", nil
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to copy TTF fonts to output path %q", fontsDirectoryPath)
	}

	// Copy extra files
	docsDirectoryPath := path.Join(shareOutputFilesystemPath, "docs", "fonts-dejavu")
	err = copy.Copy(buildOutputDirectoryPath, docsDirectoryPath, copy.Options{
		PreserveTimes:     true,
		PermissionControl: copy.AddPermission(0644),
		Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
			return path.Ext(src) == ".ttf", nil
		},
	})
	if err != nil {
		return trace.Wrap(err, "failed to copy extra doc files to output path %q, docsDirectoryPath")
	}

	// Update file ownership and permissions
	err = filepath.WalkDir(z.OutputDirectoryPath, func(fsPath string, fsEntry fs.DirEntry, err error) error {
		if err != nil {
			return trace.Wrap(err, "failed to walk dir %q", fsPath)
		}

		err = os.Chown(fsPath, 0, 0)
		if err != nil {
			return trace.Wrap(err, "failed to update owner to root:root on output file %q", fsPath)
		}

		permissions := fs.FileMode(0644)
		if fsEntry.IsDir() {
			permissions = fs.FileMode(0755)
		}

		err = os.Chmod(fsPath, permissions)
		if err != nil {
			return trace.Wrap(err, "failed to update permisions to %o on output file %q", int32(permissions), fsPath)
		}

		return nil
	})
	if err != nil {
		return trace.Wrap(err, "failed to update output file ownership and permissions")
	}

	return nil
}
