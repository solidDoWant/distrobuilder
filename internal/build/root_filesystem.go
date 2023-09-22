package build

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gravitational/trace"
)

type RootFilesystem struct {
	FilesystemOutputBuilder
}

func (rfs *RootFilesystem) CheckHostRequirements() error {
	return nil
}

func (rfs *RootFilesystem) Build(ctx context.Context) error {
	outputDirectory, err := setupOutputDirectory(rfs.OutputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to setup output directory")
	}
	rfs.OutputDirectoryPath = outputDirectory.Path

	rootFSTree := rfs.getFilesystemStructure()
	err = rootFSTree.Create(rfs.OutputDirectoryPath, "")
	if err != nil {
		return trace.Wrap(err, "failed to create rootfs tree at %q", rfs.OutputDirectoryPath)
	}

	users, groups := rfs.getUsersAndGroups()

	err = users.WritePasswdFile(rfs.OutputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to write user passwd file to rootfs at %q", rfs.OutputDirectoryPath)
	}

	err = groups.WriteGroupFile(rfs.OutputDirectoryPath)
	if err != nil {
		return trace.Wrap(err, "failed to write group file to rootfs at %q", rfs.OutputDirectoryPath)
	}

	return nil
}

func (rfs *RootFilesystem) VerifyBuild(context.Context) error {
	// TODO verify what, if any, checks need to be performed here
	return nil
}

func (rfs *RootFilesystem) createGroups(rootFSPath string) error {
	filePath := path.Join(rootFSPath, "etc", "group")

	fileContents := ""

	err := os.WriteFile(path.Join(rootFSPath, "etc", "group"), []byte(fileContents), 0644)
	if err != nil {
		return trace.Wrap(err, "failed to create group file at %q", filePath)
	}

	return nil
}

type systemGroup struct {
	Name  string
	ID    uint32
	Users []string
}

func (sg *systemGroup) GetGroupFileEntry() string {
	return fmt.Sprintf("%s:x:%d:%s", sg.Name, sg.ID, strings.Join(sg.Users, ","))
}

type systemGroups []*systemGroup

func (sg systemGroups) GetGroupFile() string {
	fileContents := ""

	slices.SortFunc(sg, func(a, b *systemGroup) int { return cmp.Compare(a.ID, b.ID) })

	for _, group := range sg {
		if group == nil {
			continue
		}

		fileContents += group.GetGroupFileEntry()
		fileContents += "\n"
	}

	return fileContents
}

func (sg systemGroups) WriteGroupFile(rootFSPath string) error {
	filePath := path.Join(rootFSPath, "etc", "group")
	slog.Info("Writing group file", "path", filePath)

	fileContents := sg.GetGroupFile()

	err := os.WriteFile(filePath, []byte(fileContents), 0644)
	if err != nil {
		return trace.Wrap(err, "failed to create group file at %q", filePath)
	}

	return nil
}

type systemUser struct {
	Name              string
	ID                uint32
	PrimaryGroupID    uint32
	Info              string
	HomeDirectoryPath string
	ShellPath         string
}

func (su *systemUser) GetPasswdFileEntry() string {
	return fmt.Sprintf("%s:x:%d:%d:%s:%s:%s", su.Name, su.ID, su.PrimaryGroupID, su.Info, su.HomeDirectoryPath, su.ShellPath)
}

type systemUsers []*systemUser

func (su systemUsers) GetPasswdFile() string {
	fileContents := ""

	slices.SortFunc(su, func(a, b *systemUser) int { return cmp.Compare(a.ID, b.ID) })

	for _, user := range su {
		if user == nil {
			continue
		}

		fileContents += user.GetPasswdFileEntry()
		fileContents += "\n"
	}

	return fileContents
}

func (su systemUsers) WritePasswdFile(rootFSPath string) error {
	filePath := path.Join(rootFSPath, "etc", "passwd")
	slog.Info("Creating passwd file", "path", filePath)

	fileContents := su.GetPasswdFile()

	err := os.WriteFile(filePath, []byte(fileContents), 0644)
	if err != nil {
		return trace.Wrap(err, "failed to create passwd file at %q", filePath)
	}

	return nil
}

func (rfs *RootFilesystem) getUsersAndGroups() (systemUsers, systemGroups) {
	return []*systemUser{
			{
				Name:              "root",
				ID:                0,
				PrimaryGroupID:    0,
				Info:              "root",
				HomeDirectoryPath: "/root",
				ShellPath:         "/bin/bash",
			},
			{
				Name:              "nobody",
				ID:                65534,
				PrimaryGroupID:    65534,
				Info:              "nobody",
				HomeDirectoryPath: "/nonexistent",
				ShellPath:         "/usr/sbin/nologin",
			},
		}, []*systemGroup{
			{
				Name: "root",
				ID:   0,
			},
			{
				Name: "nogroup",
				ID:   65534,
			},
		}
}

type rootFSObject struct {
	Name          string
	Permissions   uint16
	ChildObjects  []*rootFSObject
	SymlinkTarget string // If set them the object will be symlinked to the specified path
	SetStickyBit  bool   // If set the object will be marked as sticky bit, invalid on symlinks
	SetGroupId    bool   // If set the object will have the setgid bit enabled, invalid on symlinks
	GroupID       int
	UserID        int
}

func (rfso *rootFSObject) GetFileMode() fs.FileMode {
	fileMode := fs.FileMode(rfso.Permissions)
	if rfso.SetStickyBit {
		fileMode |= fs.ModeSticky
	}
	if rfso.SetGroupId {
		fileMode |= fs.ModeSetgid
	}

	return fileMode
}

func (rfso *rootFSObject) Create(treeRootPath, parentRelativetPath string) error {
	selfPath := path.Join(parentRelativetPath, rfso.Name)
	err := rfso.createSelf(treeRootPath, selfPath)
	if err != nil {
		return trace.Wrap(err, "failed to create object at %q relative to root %q", selfPath, treeRootPath)
	}

	// Create each child object
	for _, childObject := range rfso.ChildObjects {
		err := childObject.Create(treeRootPath, selfPath)
		if err != nil {
			return trace.Wrap(err, "failed to create child object %q", childObject.Name)
		}
	}

	return nil
}

func (rfso *rootFSObject) createSelf(treeRootPath, selfPath string) error {
	selfAbsolutePath := path.Join(treeRootPath, selfPath)
	slog.Debug("creating filesystem object", "path", selfAbsolutePath)

	if rfso.SymlinkTarget == "" {
		if selfAbsolutePath != treeRootPath {
			// Top level entry should already exist
			err := os.Mkdir(selfAbsolutePath, rfso.GetFileMode())
			if err != nil {
				return trace.Wrap(err, "failed to create directory %q", selfPath)
			}
		}

		err := os.Chown(selfAbsolutePath, rfso.UserID, rfso.GroupID)
		if err != nil {
			return trace.Wrap(err, "failed to set ownership of %q to %d:%d", selfAbsolutePath, rfso.UserID, rfso.GroupID)
		}

		return nil
	}

	target := rfso.SymlinkTarget
	// If symlink is relative to the root of the new filesystem
	if target[0] == os.PathSeparator {
		relativeTarget, err := filepath.Rel(path.Dir(selfPath), target)
		if err != nil {
			return trace.Wrap(err, "failed to get link target %q relative to link path %q", target, selfPath)
		}

		target = relativeTarget
	}

	err := os.Symlink(target, selfAbsolutePath)
	if err != nil {
		return trace.Wrap(err, "failed to create symlink %q", selfPath)
	}

	return nil
}

func (rfs *RootFilesystem) getFilesystemStructure() *rootFSObject {
	// Architecture independent data files.
	shareFSObjects := &rootFSObject{
		Name:        "share",
		Permissions: 0755,
		ChildObjects: []*rootFSObject{
			// Word lists.
			{
				Name:        "dict",
				Permissions: 0755,
			},
			// Miscellaneous documentation.
			{
				Name:        "doc",
				Permissions: 0755,
			},
			// Game data.
			{
				Name:        "games",
				Permissions: 0755,
			},
			// GNU info system data.
			{
				Name:        "info",
				Permissions: 0755,
			},
			// Locale information.
			{
				Name:        "locale",
				Permissions: 0755,
			},
			// Manual pages.
			rfs.getManFSObjects(false),
			// Miscellaneous data.
			{
				Name:        "misc",
				Permissions: 0755,
			},
			// Timezone information.
			{
				Name:        "zoneinfo",
				Permissions: 0755,
			},
		},
	}

	// Create directories in accordance to the Filesystem Hierarchy Standard
	// See https://refspecs.linuxfoundation.org/fhs.shtml for details
	root := &rootFSObject{
		Name:        "/",
		Permissions: 0755,
		ChildObjects: []*rootFSObject{
			// Essential user command binaries that are required when no other filesystems are mounted.
			{
				// The /usr will be mounted (if needed) by the initramfs.
				// Therefore none of the commands required by the FHS
				// are needed to mount other filesystems. Symlinking
				// to /usr simplifies the filesystem hirarchy while
				// still being compliant.
				Name:          "bin",
				SymlinkTarget: "/usr/bin",
			},
			// Files required during the boot process for the bootloader.
			{
				Name:        "boot",
				Permissions: 0755,
			},
			// Device files; interfaces to devices and their drivers.
			{
				Name:        "dev",
				Permissions: 0755,
				ChildObjects: []*rootFSObject{
					// Shared memory directory, expected to be backed by a tempfs.
					{
						Name: "shm",
						// Symlinked to reduce the number of tempfs mounts required.
						SymlinkTarget: "/run/shm",
					},
				},
			},
			// Configuration files that are host-specific.
			{
				Name:        "etc",
				Permissions: 0755,
				ChildObjects: []*rootFSObject{
					// Configuration for local binaries
					{
						Name:        "local",
						Permissions: 0755,
					},
					// Dynamic filesystem information, symlinked for historical purposes.
					{
						Name:          "mtab",
						SymlinkTarget: "/proc/mounts",
					},
					// Configuration files for software pacakges in /opt.
					{
						Name:        "opt",
						Permissions: 0755,
					},
				},
			},
			// Parent directory of user home directories.
			{
				Name:        "home",
				Permissions: 0755,
				// TODO consider implementing https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s08.html#idm236092745632
			},
			// Shared libraries needed to boot the system that are required for booting the system.
			{
				// As with /bin, these files are non-essential due to the initramfs.
				// The directory is symlinked to /usr/lib to simplify the filesystem hirarchy.
				Name:          "lib",
				SymlinkTarget: "/usr/lib",
			},
			// Mount ponits for removable media.
			{
				Name:        "media",
				Permissions: 0755,
			},
			// Mount point for temporary filesystem mounts.
			{
				Name:        "mnt",
				Permissions: 0755,
			},
			// Optional software packages.
			{
				Name:        "opt",
				Permissions: 0755,
			},
			// Process information virtual filesystem.
			{
				Name:        "proc",
				Permissions: 0555,
			},
			// Home directory for root user.
			{
				Name:        "root",
				Permissions: 0700,
			},
			// Runtime data for the system that has been generated since the last boot.
			{
				// This should be a tempfs, but directories are added here to ensure that
				// the paths exist even if not created during rootfs mount.
				Name:        "run",
				Permissions: 0755,
				ChildObjects: []*rootFSObject{
					// Per-program lock files.
					{
						Name:         "lock",
						Permissions:  0777,
						SetStickyBit: true,
					},
					// Shared memory directory, expected to be backed by a tempfs.
					{
						Name:         "shm",
						Permissions:  0777,
						SetStickyBit: true,
					},
					// Temporary scratch space for users and programs.
					{
						Name:         "tmp",
						Permissions:  0777,
						SetStickyBit: true,
					},
				},
			},
			// System administration specific binaries that are required for booting the system.
			{
				// As with /bin, these files are non-essential due to the initramfs.
				// The directory is symlinked to /usr/lib to simplify the filesystem hirarchy.
				Name:          "sbin",
				SymlinkTarget: "/usr/sbin",
			},
			// Data files for services.
			{
				Name:        "srv",
				Permissions: 0755,
			},
			// System information virtual filesystem.
			{
				Name:        "sys",
				Permissions: 0555,
			},
			// Temporary scratch space for users and programs.
			{
				// This directory is symlinked to /run/tmp to reduce the number of tempfs mount points.
				Name:          "tmp",
				SymlinkTarget: "/run/tmp",
			},
			// Sharable, read-only data
			{
				Name:        "usr",
				Permissions: 0755,
				ChildObjects: []*rootFSObject{
					// Commands that are not essential to boot, used by ordinary users.
					{
						Name:        "bin",
						Permissions: 0755,
					},
					// Games and educational programs
					{
						Name:        "games",
						Permissions: 0755,
					},
					// System header files for C.
					{
						Name: "include",
					},
					// Object files and libraries, as well as internal binaries such as the dynamic loader.
					{
						Name:        "lib",
						Permissions: 0755,
						ChildObjects: []*rootFSObject{
							// Linux kernel modules, required by FHS for /lib.
							{
								Name:        "modules",
								Permissions: 0755,
							},
						},
					},
					// Internal binaries not intended to be ran directly by users or scripts.
					{
						Name:        "libexec",
						Permissions: 0755,
					},
					// Locally installed software that will not be overwritten by system upgrades.
					{
						Name:        "local",
						Permissions: 0755,
						// These are all local versions of the directories in /bin unless otherwise specified.
						ChildObjects: []*rootFSObject{
							{
								Name:        "bin",
								Permissions: 0755,
							},
							{
								// Symlinked to simplify the filesystem hierarchy.
								Name:          "etc",
								SymlinkTarget: "/etc/local",
							},
							{
								Name:        "games",
								Permissions: 0755,
							},
							{
								Name:        "include",
								Permissions: 0755,
							},
							{
								Name:        "lib",
								Permissions: 0755,
							},
							{
								// Symlinked to simplify the filesystem hierarchy.
								Name:          "man",
								SymlinkTarget: "share/man",
							},
							{
								Name:        "sbin",
								Permissions: 0755,
							},
							shareFSObjects,
							{
								Name:        "src",
								Permissions: 0755,
							},
						},
					},
					// Commands that are not essential to boot, used by system administrators.
					{
						Name:        "sbin",
						Permissions: 0755,
					},
					shareFSObjects,
					// Source code reference files.
					{
						Name:        "src",
						Permissions: 0755,
					},
				},
			},
			// Variable data files, mostly persisting across reboots.
			{
				Name:        "var",
				Permissions: 0755,
				ChildObjects: []*rootFSObject{
					// Application backup data.
					{
						Name:        "backups",
						Permissions: 0755,
					},
					// Application cache data.
					{
						Name:        "cache",
						Permissions: 0755,
						ChildObjects: []*rootFSObject{
							// Cache for formatted manual pages.
							rfs.getManFSObjects(true),
						},
					},
					// System crash dumps.
					{
						Name:         "crash",
						Permissions:  0777,
						SetStickyBit: true,
					},
					// Variable game data
					{
						Name:        "games",
						Permissions: 0755,
					},
					// Variable state information.
					{
						Name:        "lib",
						Permissions: 0755,
						ChildObjects: []*rootFSObject{
							// State files that do not need a subdirectory.
							{
								Name:        "misc",
								Permissions: 0755,
							},
						},
					},
					// Variable data for /usr/local.
					{
						Name:         "local",
						Permissions:  0775,
						SetStickyBit: true,
					},
					// Lock files.
					{
						// Symlinked to reduce the number of tempfs mounts.
						Name:          "lock",
						SymlinkTarget: "/run/lock",
					},
					// Log files and directories.
					{
						Name: "log",
					},
					// User mailbox files
					{
						Name:         "mail",
						Permissions:  0775,
						SetStickyBit: true,
					},
					// Variable data for /opt.
					{
						Name:        "opt",
						Permissions: 0755,
					},
					// Variable data for running processes, not persisted across reboots.
					{
						Name:          "run",
						SymlinkTarget: "/run",
					},
					// Application data awaiting processing.
					{
						Name:        "spool",
						Permissions: 0755,
						ChildObjects: []*rootFSObject{
							// Mail, to preserve backwards compatibility with older programs.
							{
								Name:          "mail",
								SymlinkTarget: "/var/mail",
							},
							// Variable data for cron jobs and at programs.
							{
								Name:        "cron",
								Permissions: 0755,
							},
						},
					},
					// Temporary data, preserved across reboots.
					{
						Name:         "tmp",
						Permissions:  0777,
						SetStickyBit: true,
					},
				},
			},
		},
	}

	return root
}

func (rfs *RootFilesystem) getManFSObjects(isCacheDir bool) *rootFSObject {
	sectionPrefix := "man"
	if isCacheDir {
		sectionPrefix = "cat"
	}

	sectionDirNames := make([]string, 8)
	for i := range sectionDirNames {
		sectionDirNames[i] = fmt.Sprintf("%s%d", sectionPrefix, i+1)
	}

	return &rootFSObject{
		Name:        "man",
		Permissions: 0755,
		ChildObjects: []*rootFSObject{
			// English documentation.
			{
				Name:          "en",
				SymlinkTarget: "en_US",
			},
			// English/US documentation.
			{
				Name:        "en_US",
				Permissions: 0755,
				ChildObjects: []*rootFSObject{
					// User program documention.
					{
						Name:        sectionDirNames[0],
						Permissions: 0755,
					},
					// System call documentation.
					{
						Name:        sectionDirNames[1],
						Permissions: 0755,
					},
					// Library call documentation.
					{
						Name:        sectionDirNames[2],
						Permissions: 0755,
					},
					// Special file documentation.
					{
						Name:        sectionDirNames[3],
						Permissions: 0755,
					},
					// File format documentation.
					{
						Name:        sectionDirNames[4],
						Permissions: 0755,
					},
					// Game documentation.
					{
						Name:        sectionDirNames[5],
						Permissions: 0755,
					},
					// Miscellaneous documentation.
					{
						Name:        sectionDirNames[6],
						Permissions: 0755,
					},
					// System adminisration documentation.
					{
						Name:        sectionDirNames[7],
						Permissions: 0755,
					},
				},
			},
			// User program documention.
			{
				Name:          sectionDirNames[0],
				SymlinkTarget: path.Join("en", sectionDirNames[0]),
			},
			// System call documentation.
			{
				Name:          sectionDirNames[1],
				SymlinkTarget: path.Join("en", sectionDirNames[1]),
			},
			// Library call documentation.
			{
				Name:          sectionDirNames[2],
				SymlinkTarget: path.Join("en", sectionDirNames[2]),
			},
			// Special file documentation.
			{
				Name:          sectionDirNames[3],
				SymlinkTarget: path.Join("en", sectionDirNames[3]),
			},
			// File format documentation.
			{
				Name:          sectionDirNames[4],
				SymlinkTarget: path.Join("en", sectionDirNames[4]),
			},
			// Game documentation.
			{
				Name:          sectionDirNames[5],
				SymlinkTarget: path.Join("en", sectionDirNames[5]),
			},
			// Miscellaneous documentation.
			{
				Name:          sectionDirNames[6],
				SymlinkTarget: path.Join("en", sectionDirNames[6]),
			},
			// System adminisration documentation.
			{
				Name:          sectionDirNames[7],
				SymlinkTarget: path.Join("en", sectionDirNames[7]),
			},
		},
	}
}
