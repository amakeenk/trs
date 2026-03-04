package trash

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// HomeTrashDir returns the default XDG trash directory
func HomeTrashDir() (string, error) {
	// Check XDG_DATA_HOME first
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "Trash"), nil
	}

	// Fall back to ~/.local/share/Trash
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return filepath.Join(home, ".local", "share", "Trash"), nil
}

// VolumeTrashDir returns the trash directory for a given path's volume
// For cross-device files, use $VOLUME/.Trash-$UID/
func VolumeTrashDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("get absolute path: %w", err)
	}

	// Get the mount point of the path
	mountPoint, err := getMountPoint(absPath)
	if err != nil {
		return "", fmt.Errorf("get mount point: %w", err)
	}

	// Check if same as home filesystem
	homeTrash, err := HomeTrashDir()
	if err != nil {
		return "", err
	}

	homeMount, err := getMountPoint(homeTrash)
	if err != nil {
		return "", fmt.Errorf("get home mount point: %w", err)
	}

	if mountPoint == homeMount {
		// Same filesystem, use home trash
		return homeTrash, nil
	}

	// Different filesystem, use volume trash
	uid := os.Getuid()
	volumeTrash := filepath.Join(mountPoint, fmt.Sprintf(".Trash-%d", uid))

	return volumeTrash, nil
}

// getMountPoint returns the mount point for a given path
func getMountPoint(path string) (string, error) {
	// Find first existing parent if path doesn't exist
	for {
		_, err := os.Stat(path)
		if err == nil {
			break
		}
		if os.IsNotExist(err) {
			parent := filepath.Dir(path)
			if parent == path {
				// Reached root
				return "/", nil
			}
			path = parent
			continue
		}
		return "", fmt.Errorf("stat %s: %w", path, err)
	}

	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}

	// Walk up the directory tree until we find a different device
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return current, nil
		}

		var parentStat syscall.Stat_t
		if err := syscall.Stat(parent, &parentStat); err != nil {
			return "", fmt.Errorf("stat parent %s: %w", parent, err)
		}

		if parentStat.Dev != stat.Dev {
			// Different device, current is the mount point
			return current, nil
		}

		current = parent
	}
}

// SameFilesystem checks if two paths are on the same filesystem
func SameFilesystem(path1, path2 string) (bool, error) {
	var stat1, stat2 syscall.Stat_t

	if err := syscall.Stat(path1, &stat1); err != nil {
		return false, fmt.Errorf("stat %s: %w", path1, err)
	}

	if err := syscall.Stat(path2, &stat2); err != nil {
		return false, fmt.Errorf("stat %s: %w", path2, err)
	}

	return stat1.Dev == stat2.Dev, nil
}

// EnsureTrashDir creates the trash directory structure if it doesn't exist
func EnsureTrashDir(trashDir string) error {
	// Security check: verify trashDir is not a symlink
	if fi, err := os.Lstat(trashDir); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("trash directory cannot be a symlink: %s", trashDir)
		}
		// Verify ownership
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			if int(stat.Uid) != os.Getuid() {
				return fmt.Errorf("trash directory not owned by current user: %s", trashDir)
			}
		}
		// Verify permissions (should be 0700)
		if fi.Mode().Perm() != 0700 {
			return fmt.Errorf("trash directory has insecure permissions: %s (expected 0700, got %03o)", trashDir, fi.Mode().Perm())
		}
	}

	dirs := []string{
		trashDir,
		filepath.Join(trashDir, "files"),
		filepath.Join(trashDir, "info"),
	}

for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create trash directory %s: %w", dir, err)
		}
		// Verify the created directory is not a symlink
		if fi, err := os.Lstat(dir); err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("trash directory cannot be a symlink: %s", dir)
			}
		}
	}

	return nil
}

// GetTrashDirForPath returns the appropriate trash directory for a given path
func GetTrashDirForPath(path string) (string, error) {
	trashDir, err := VolumeTrashDir(path)
	if err != nil {
		return "", err
	}

	if err := EnsureTrashDir(trashDir); err != nil {
		return "", err
	}

	return trashDir, nil
}
