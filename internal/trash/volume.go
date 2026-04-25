package trash

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var mountsFilePath = "/proc/mounts"

// SetMountsFilePath sets the path for mount points file (primarily for testing)
func SetMountsFilePath(path string) {
	mountsFilePath = path
}

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

	// Check for .Trash/$UID (XDG Spec)
	dotTrash := filepath.Join(mountPoint, ".Trash")
	if fi, err := os.Lstat(dotTrash); err == nil && fi.IsDir() {
		// Ensure it has the sticky bit set and is not a symlink
		if fi.Mode()&os.ModeSticky != 0 && fi.Mode()&os.ModeSymlink == 0 {
			return filepath.Join(dotTrash, strconv.Itoa(uid)), nil
		}
	}

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

// checkSecureDir verifies that a directory exists, is owned by the current user,
// has correct permissions, and is not a symlink.
func checkSecureDir(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if !fi.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("directory cannot be a symlink: %s", path)
	}

	// Verify ownership
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		if int(stat.Uid) != os.Getuid() {
			// For network/virtual filesystems, UID mismatch is common due to mapping.
			// If the filesystem is non-local, we allow it because the user
			// has effective access and it's their intended trash directory.
			fstype, _ := getFsType(path)
			if isLocalFilesystem(fstype) {
				return fmt.Errorf("directory not owned by current user: %s (owned by UID %d)", path, stat.Uid)
			}
		}
	}

	// Verify permissions (strictly 0700)
	if fi.Mode().Perm() != 0700 {
		return fmt.Errorf("directory has insecure permissions: %s (expected 0700, got %03o)", path, fi.Mode().Perm())
	}

	return nil
}

// getFsType returns the filesystem type for a given path
func getFsType(path string) (string, error) {
	mountPoint, err := getMountPoint(path)
	if err != nil {
		return "", err
	}

	mounts, err := os.Open(mountsFilePath)
	if err != nil {
		return "", err
	}
	defer mounts.Close()

	scanner := bufio.NewScanner(mounts)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[1] == mountPoint {
			return fields[2], nil
		}
	}

	return "unknown", nil
}

// EnsureTrashDir creates the trash directory structure if it doesn't exist
func EnsureTrashDir(trashDir string) error {
	dirs := []string{
		trashDir,
		filepath.Join(trashDir, "files"),
		filepath.Join(trashDir, "info"),
	}

	for _, dir := range dirs {
		// Use 0700 to ensure only owner can access during creation
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create trash directory %s: %w", dir, err)
		}

		// MANDATORY: Verify security AFTER creation to prevent TOCTOU attacks
		if err := checkSecureDir(dir); err != nil {
			return fmt.Errorf("security check failed for %s: %w", dir, err)
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

// nonLocalPrefixes lists filesystem type prefixes that indicate network/virtual mounts.
// Any fstype starting with one of these prefixes is considered non-local.
var nonLocalPrefixes = []string{
	"fuse", "nfs", "cifs", "sshfs", "davfs", "glusterfs", "ceph", "autofs",
}

// nonLocalExact lists exact filesystem type names that are virtual or non-local.
var nonLocalExact = map[string]bool{
	"tmpfs": true, "devtmpfs": true, "devpts": true, "proc": true,
	"sysfs": true, "debugfs": true, "securityfs": true, "cgroup": true,
	"cgroup2": true, "pstore": true, "bpf": true, "tracefs": true,
	"configfs": true, "hugetlbfs": true, "mqueue": true, "rpc_pipefs": true,
	"binfmt_misc": true, "squashfs": true, "iso9660": true, "overlay": true,
}

// isLocalFilesystem returns true if the filesystem type is a local disk filesystem
// that is safe to stat without risk of hanging.
func isLocalFilesystem(fstype string) bool {
	if nonLocalExact[fstype] {
		return false
	}
	for _, prefix := range nonLocalPrefixes {
		if strings.HasPrefix(fstype, prefix) {
			return false
		}
	}
	return true
}

// statWithTimeout calls os.Stat with a timeout to prevent blocking on
// unresponsive filesystems (e.g. hard NFS mounts, autofs triggers).
func statWithTimeout(path string, timeout time.Duration) (os.FileInfo, error) {
	type result struct {
		fi  os.FileInfo
		err error
	}
	// Use buffered channel to prevent goroutine leak if we timeout
	ch := make(chan result, 1)
	go func() {
		fi, err := os.Stat(path)
		ch <- result{fi, err}
	}()
	select {
	case res := <-ch:
		return res.fi, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("stat %s: timeout after %v", path, timeout)
	}
}

// GetAllTrashDirs returns all trash directories that exist
// This includes home trash and any volume trash directories
func GetAllTrashDirs() ([]string, error) {
	var dirs []string
	uid := os.Getuid()

	// Always add home trash first
	homeTrash, err := HomeTrashDir()
	if err != nil {
		return nil, err
	}
	dirs = append(dirs, homeTrash)

	// Parse mount points from /proc/mounts
	mounts, err := os.Open(mountsFilePath)
	if err != nil {
		// If /proc/mounts doesn't exist, just return home trash
		return dirs, nil
	}
	defer mounts.Close()

	// Track seen mount points to avoid duplicates (bind mounts)
	seen := make(map[string]bool)
	homeMount, err := getMountPoint(homeTrash)
	if err == nil {
		seen[homeMount] = true
	}

	scanner := bufio.NewScanner(mounts)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mountPoint := fields[1]
		fstype := fields[2]

		// Skip virtual filesystems that definitely won't have trash to avoid noise
		if nonLocalExact[fstype] && fstype != "tmpfs" {
			continue
		}

		// Skip if already seen (handles bind mounts)
		if seen[mountPoint] {
			continue
		}
		seen[mountPoint] = true

		// Check for .Trash/$UID directory (XDG Spec) with timeout
		dotTrash := filepath.Join(mountPoint, ".Trash")
		if fi, err := statWithTimeout(dotTrash, 2*time.Second); err == nil && fi.IsDir() {
			if fi.Mode()&os.ModeSticky != 0 && fi.Mode()&os.ModeSymlink == 0 {
				volumeTrashXDG := filepath.Join(dotTrash, strconv.Itoa(uid))
				if fi2, err2 := statWithTimeout(volumeTrashXDG, 2*time.Second); err2 == nil && fi2.IsDir() {
					dirs = append(dirs, volumeTrashXDG)
				}
			}
		}

		// Check for .Trash-$UID volume trash directory with timeout
		volumeTrash := filepath.Join(mountPoint, fmt.Sprintf(".Trash-%d", uid))
		if fi, err := statWithTimeout(volumeTrash, 2*time.Second); err == nil && fi.IsDir() {
			dirs = append(dirs, volumeTrash)
		}
	}

	return dirs, nil
}
