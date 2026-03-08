package trash

import (
	"fmt"
	"os"
)

// TrashRoot provides os.Root-based access to trash directory.
// All operations are traversal-resistant - symlinks and ".." cannot escape the root.
type TrashRoot struct {
	root *os.Root
}

// OpenTrashRoot opens a trash directory for os.Root-based operations.
// The trashDir must be an absolute path to the trash directory root
// (e.g., ~/.local/share/Trash or /volume/.Trash-$UID).
func OpenTrashRoot(trashDir string) (*TrashRoot, error) {
	root, err := os.OpenRoot(trashDir)
	if err != nil {
		return nil, fmt.Errorf("open trash root %s: %w", trashDir, err)
	}
	return &TrashRoot{root: root}, nil
}

// Close closes the trash root.
func (tr *TrashRoot) Close() error {
	return tr.root.Close()
}

// FilesPath returns the relative path for a file in the files/ directory.
func (tr *TrashRoot) FilesPath(name string) string {
	return "files/" + name
}

// InfoPath returns the relative path for a trashinfo file in the info/ directory.
func (tr *TrashRoot) InfoPath(name string) string {
	return "info/" + name + ".trashinfo"
}

// LstatFiles returns FileInfo for a file in files/ directory.
// Uses os.Root.Lstat - traversal-resistant, does not follow symlinks.
func (tr *TrashRoot) LstatFiles(name string) (os.FileInfo, error) {
	return tr.root.Lstat(tr.FilesPath(name))
}

// LstatInfo returns FileInfo for a trashinfo file in info/ directory.
func (tr *TrashRoot) LstatInfo(name string) (os.FileInfo, error) {
	return tr.root.Lstat(tr.InfoPath(name))
}

// OpenFiles opens a file in files/ directory for reading.
func (tr *TrashRoot) OpenFiles(name string) (*os.File, error) {
	return tr.root.Open(tr.FilesPath(name))
}

// OpenInfo opens a trashinfo file in info/ directory for reading.
func (tr *TrashRoot) OpenInfo(name string) (*os.File, error) {
	return tr.root.Open(tr.InfoPath(name))
}

// CreateFiles creates a new file in files/ directory.
func (tr *TrashRoot) CreateFiles(name string) (*os.File, error) {
	return tr.root.Create(tr.FilesPath(name))
}

// CreateInfo creates a new trashinfo file in info/ directory.
func (tr *TrashRoot) CreateInfo(name string) (*os.File, error) {
	return tr.root.Create(tr.InfoPath(name))
}

// RemoveFiles removes a file from files/ directory.
func (tr *TrashRoot) RemoveFiles(name string) error {
	return tr.root.Remove(tr.FilesPath(name))
}

// RemoveInfo removes a trashinfo file from info/ directory.
func (tr *TrashRoot) RemoveInfo(name string) error {
	return tr.root.Remove(tr.InfoPath(name))
}

// ReadDirFiles reads the files/ directory.
func (tr *TrashRoot) ReadDirFiles() ([]os.DirEntry, error) {
	f, err := tr.root.Open("files")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}

// MkdirFiles creates files/ directory.
func (tr *TrashRoot) MkdirFiles() error {
	return tr.root.Mkdir("files", 0700)
}

// MkdirInfo creates info/ directory.
func (tr *TrashRoot) MkdirInfo() error {
	return tr.root.Mkdir("info", 0700)
}


// RemoveAllFiles recursively removes a file or directory from files/.
// Unlike os.RemoveAll, this uses os.Root operations which are
// traversal-resistant - symlinks cannot escape the root.
func (tr *TrashRoot) RemoveAllFiles(name string) error {
	return tr.removeAll(tr.FilesPath(name))
}

// removeAll is the internal recursive implementation.
func (tr *TrashRoot) removeAll(relPath string) error {
	// Use Lstat to check type without following symlinks
	info, err := tr.root.Lstat(relPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return err
	}

	// If it's a file or symlink, just remove it
	if !info.IsDir() {
		return tr.root.Remove(relPath)
	}

	// For directories, we need to remove contents first
	// Open the directory using root (traversal-resistant)
	// Open the directory using root (traversal-resistant)
	f, err := tr.root.Open(relPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read directory entries
	entries, err := f.ReadDir(-1)
	if err != nil {
		return err
	}

	// Recursively remove each entry
	for _, entry := range entries {
		entryPath := relPath + "/" + entry.Name()

		if err := tr.removeAll(entryPath); err != nil {
			return err
		}
	}

	// Now remove the empty directory
	return tr.root.Remove(relPath)
}