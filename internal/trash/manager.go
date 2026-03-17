package trash

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// TrashItem represents a file in the trash
type TrashItem struct {
	Name         string    // Name in trash
	OriginalPath string    // Original absolute path
	DeletionDate time.Time // When it was trashed
	Size         int64     // Size in bytes
	IsDir        bool      // Is directory
	TrashDir     string    // Trash directory containing this item
}

type ItemType int

const (
	ItemTypeFile ItemType = iota
	ItemTypeDirectory
	ItemTypeSymlink
)

type VerboseCallback func(path string, itemType ItemType)

// Manager handles trash operations
type Manager struct {
	homeTrash       string
	verboseCallback VerboseCallback
}

// NewManager creates a new trash manager
func NewManager() (*Manager, error) {
	homeTrash, err := HomeTrashDir()
	if err != nil {
		return nil, err
	}

	return &Manager{homeTrash: homeTrash}, nil
}

func (m *Manager) SetVerboseCallback(cb VerboseCallback) {
	m.verboseCallback = cb
}

// Move moves a file or directory to the trash
func (m *Manager) Move(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	// Check if path exists (use Lstat for symlinks)
	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", absPath)
		}
		return fmt.Errorf("stat %s: %w", absPath, err)
	}

	var itemType ItemType
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		itemType = ItemTypeSymlink
	case info.IsDir():
		itemType = ItemTypeDirectory
	default:
		itemType = ItemTypeFile
	}

	// Validate filename to prevent path traversal
	baseName := filepath.Base(absPath)
	if err := validateFileName(baseName); err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}

	// Get appropriate trash directory
	trashDir, err := GetTrashDirForPath(absPath)
	if err != nil {
		return fmt.Errorf("get trash directory: %w", err)
	}

	// Create trashinfo
	ti := &TrashInfo{
		Path:         absPath,
		DeletionDate: time.Now(),
	}

	// Retry loop to handle TOCTOU race conditions
	maxAttempts := 1000
	for attempt := 0; attempt < maxAttempts; attempt++ {
		trashName, err := m.resolveNameConflict(trashDir, baseName)
		if err != nil {
			return fmt.Errorf("resolve name conflict: %w", err)
		}

		infoPath := TrashInfoPath(trashDir, trashName)
		destPath := FilesPath(trashDir, trashName)

		// Atomically create trashinfo with exclusive flag
		if err := ti.WriteExclusive(infoPath); err != nil {
			if os.IsExist(err) {
				// Race condition: another process created it, retry with new name
				continue
			}
			return fmt.Errorf("write trashinfo: %w", err)
		}

		// Move the file/directory
		if err := m.movePathVerbose(absPath, destPath, info, itemType); err != nil {
			// Clean up trashinfo on failure
			os.Remove(infoPath)
			return fmt.Errorf("move to trash: %w", err)
		}

		return nil
	}

	return fmt.Errorf("max attempts (%d) exceeded", maxAttempts)
}

// movePath handles moving a file/directory, including cross-device moves
func (m *Manager) movePath(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	if isCrossDeviceError(err) {
		return m.copyAndDelete(src, dst)
	}

	return err
}

func (m *Manager) movePathVerbose(src, dst string, info os.FileInfo, itemType ItemType) error {
	if m.verboseCallback != nil && info.IsDir() {
		type item struct {
			path     string
			itemType ItemType
		}
		var items []item
		filepath.Walk(src, func(path string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == src {
				return nil
			}
			if fi.IsDir() {
				items = append(items, item{path, ItemTypeDirectory})
			} else if fi.Mode()&os.ModeSymlink != 0 {
				items = append(items, item{path, ItemTypeSymlink})
			} else {
				items = append(items, item{path, ItemTypeFile})
			}
			return nil
		})
		m.verboseCallback(src, ItemTypeDirectory)
		for _, it := range items {
			m.verboseCallback(it.path, it.itemType)
		}
	}

	err := os.Rename(src, dst)
	if err == nil {
		if m.verboseCallback != nil && !info.IsDir() {
			m.verboseCallback(src, itemType)
		}
		return nil
	}

	if isCrossDeviceError(err) {
		return m.copyAndDeleteVerbose(src, dst, info)
	}

	return err
}

// copyAndDelete handles cross-device moves
func (m *Manager) copyAndDelete(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return m.copyDirAndDelete(src, dst)
	}

	if err := copyFile(src, dst); err != nil {
		os.Remove(dst)
		return err
	}

	return os.Remove(src)
}

func (m *Manager) copyAndDeleteVerbose(src, dst string, info os.FileInfo) error {
	if info == nil {
		var err error
		info, err = os.Lstat(src)
		if err != nil {
			return err
		}
	}

	if info.IsDir() {
		return m.copyDirAndDeleteVerbose(src, dst)
	}

	itemType := ItemTypeFile
	if info.Mode()&os.ModeSymlink != 0 {
		itemType = ItemTypeSymlink
	}

	if m.verboseCallback != nil {
		m.verboseCallback(src, itemType)
	}

	if err := copyFile(src, dst); err != nil {
		os.Remove(dst)
		return err
	}

	return os.Remove(src)
}

// copyDirAndDelete handles cross-device directory moves
func (m *Manager) copyDirAndDelete(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Walk and copy all files
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(target, dstPath)
		}

		return copyFile(path, dstPath)
	})

	if err != nil {
		// Cleanup partial directory on error
		os.RemoveAll(dst)
		return err
	}

	// Delete original directory (use safeRemoveAll to prevent symlink attacks)
	return safeRemoveAll(src)
}

func (m *Manager) copyDirAndDeleteVerbose(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	if m.verboseCallback != nil {
		m.verboseCallback(src, ItemTypeDirectory)
	}

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			if path != src && m.verboseCallback != nil {
				m.verboseCallback(path, ItemTypeDirectory)
			}
			return os.MkdirAll(dstPath, info.Mode())
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if m.verboseCallback != nil {
				m.verboseCallback(path, ItemTypeSymlink)
			}
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(target, dstPath)
		}

		if m.verboseCallback != nil {
			m.verboseCallback(path, ItemTypeFile)
		}

		return copyFile(path, dstPath)
	})

	if err != nil {
		os.RemoveAll(dst)
		return err
	}

	return safeRemoveAll(src)
}

// copyFile copies a single file preserving permissions
func copyFile(src, dst string) (err error) {
	// Use Lstat to avoid following symlinks
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	// Security: refuse to copy symlinks to prevent attacks
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symlink: %s", src)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}

	// Cleanup partial file on error
	defer func() {
		if err != nil {
			os.Remove(dst)
		}
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		dstFile.Close()
		return err
	}

	return dstFile.Close()
}

// resolveNameConflict adds suffix _1, _2, etc. if name already exists in trash
func (m *Manager) resolveNameConflict(trashDir, name string) (string, error) {
	// Open trash root for traversal-resistant operations
	root, err := OpenTrashRoot(trashDir)
	if err != nil {
		return "", fmt.Errorf("open trash root: %w", err)
	}
	defer root.Close()

	trashName := name
	counter := 0
	maxIterations := 10000 // Prevent DoS via infinite loop

	for {
		// Use TrashRoot - traversal-resistant via os.Root
		_, filesErr := root.LstatFiles(trashName)
		_, infoErr := root.LstatInfo(trashName)

		if os.IsNotExist(filesErr) && os.IsNotExist(infoErr) {
			return trashName, nil
		}

		counter++
		if counter > maxIterations {
			return "", fmt.Errorf("too many name conflicts (possible DoS attack)")
		}
		trashName = fmt.Sprintf("%s_%d", name, counter)
	}
}

// List returns all items in the trash from all directories
func (m *Manager) List() ([]TrashItem, error) {
	// Get all trash directories (home + volumes)
	dirs, err := GetAllTrashDirs()
	if err != nil {
		return nil, fmt.Errorf("get trash directories: %w", err)
	}

	var allItems []TrashItem
	for _, dir := range dirs {
		items, err := m.ListFromDir(dir)
		if err != nil {
			// Continue to next directory on error
			continue
		}
		allItems = append(allItems, items...)
	}

	// Sort by deletion date, newest first
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].DeletionDate.After(allItems[j].DeletionDate)
	})

	return allItems, nil
}

// ListFromDir returns items from a specific trash directory
func (m *Manager) ListFromDir(trashDir string) ([]TrashItem, error) {
	// Check if trash directory exists first (for graceful handling of non-existent dirs)
	if _, err := os.Stat(trashDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Open trash root for traversal-resistant operations
	root, err := OpenTrashRoot(trashDir)
	if err != nil {
		return nil, fmt.Errorf("open trash root: %w", err)
	}

	entries, err := root.ReadDirFiles()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read trash files: %w", err)
	}

	var items []TrashItem
	for _, entry := range entries {
		infoPath := TrashInfoPath(trashDir, entry.Name())
		ti, err := ParseTrashInfo(infoPath)
		if err != nil {
			// Skip items without valid trashinfo
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Calculate size - for directories, compute total contents size
		var size int64
		if entry.IsDir() {
			size, _ = dirSize(FilesPath(trashDir, entry.Name()))
		} else {
			size = info.Size()
		}

		items = append(items, TrashItem{
			Name:         entry.Name(),
			OriginalPath: ti.Path,
			DeletionDate: ti.DeletionDate,
			Size:         size,
			IsDir:        entry.IsDir(),
			TrashDir:     trashDir,
		})
	}

	// Sort by deletion date, newest first
	sort.Slice(items, func(i, j int) bool {
		return items[i].DeletionDate.After(items[j].DeletionDate)
	})

	return items, nil
}

// validateRestorePath validates the restore path to prevent security issues
func validateRestorePath(path string) error {
	// Must be absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("invalid restore path: must be absolute")
	}

	// Verify path doesn't escape root using relative calculation
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel("/", cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return fmt.Errorf("invalid restore path: path traversal detected")
	}

	// Also check for double-slash which could be used in attacks
	if strings.Contains(path, "//") {
		return fmt.Errorf("invalid restore path: suspicious path format")
	}

	// Block system paths
	systemPaths := []string{
		"/etc/", "/root/", "/boot/", "/dev/", "/proc/", "/sys/",
		"/bin/", "/sbin/", "/lib/", "/lib64/",
		"/usr/bin/", "/usr/sbin/", "/usr/lib/", "/usr/lib64/",
		"/usr/share/", "/usr/include/",
	}
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(cleanPath, sysPath) {
			return fmt.Errorf("refusing to restore to system path: %s", path)
		}
	}

	return nil
}

// Restore restores a file from the trash by searching all directories
func (m *Manager) Restore(trashName string, overwrite bool) error {
	// Search all trash directories for the item
	dirs, err := GetAllTrashDirs()
	if err != nil {
		return fmt.Errorf("get trash directories: %w", err)
	}

	var lastErr error
	for _, dir := range dirs {
		err := m.RestoreFromDir(dir, trashName, overwrite)
		if err == nil {
			return nil // Successfully restored
		}
		// If error is "not found in this directory", continue to next
		// If error is something else (file found but restore failed), return it
		if isNotFoundError(err) {
			continue
		}
		lastErr = err
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("file not in trash: %s", trashName)
}

// isNotFoundError checks if the error indicates the file is not in this trash directory
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for os.IsNotExist errors (trashinfo or file missing)
	if os.IsNotExist(err) {
		return true
	}
	// Check for our "read trashinfo" error wrapper
	errStr := err.Error()
	return strings.Contains(errStr, "read trashinfo") ||
		strings.Contains(errStr, "file not in trash")
}

// RestoreFromDir restores from a specific trash directory
func (m *Manager) RestoreFromDir(trashDir, trashName string, overwrite bool) error {
	infoPath := TrashInfoPath(trashDir, trashName)
	ti, err := ParseTrashInfo(infoPath)
	if err != nil {
		return fmt.Errorf("read trashinfo: %w", err)
	}

	// Security: validate path from trashinfo to prevent malicious paths
	if err := validateRestorePath(ti.Path); err != nil {
		return fmt.Errorf("invalid restore path: %w", err)
	}

	srcPath := FilesPath(trashDir, trashName)
	// Use Lstat to avoid following symlinks
	if _, err := os.Lstat(srcPath); err != nil {
		return fmt.Errorf("file not in trash: %s", trashName)
	}

	// Use Lstat to avoid symlink following attacks
	if fi, err := os.Lstat(ti.Path); err == nil {
		if !overwrite {
			return fmt.Errorf("destination exists: %s (use --force to overwrite)", ti.Path)
		}
		// Security: refuse to remove symlinks to prevent attacks
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to overwrite symlink: %s", ti.Path)
		}
		// Remove existing destination using safe removal
		if err := safeRemoveAll(ti.Path); err != nil {
			return fmt.Errorf("remove existing: %w", err)
		}
	}
	destDir := filepath.Dir(ti.Path)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	// Move back
	if err := m.movePath(srcPath, ti.Path); err != nil {
		return fmt.Errorf("restore: %w", err)
	}

	// Remove trashinfo
	os.Remove(infoPath)

	return nil
}

// Delete permanently removes a file from trash (no recovery possible)
func (m *Manager) Delete(trashName string) error {
	dirs, err := GetAllTrashDirs()
	if err != nil {
		return fmt.Errorf("get trash directories: %w", err)
	}

	var lastErr error
	for _, dir := range dirs {
		err := m.DeleteFromDir(dir, trashName)
		if err == nil {
			return nil
		}
		if isNotFoundError(err) {
			continue
		}
		lastErr = err
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("file not in trash: %s", trashName)
}

// DeleteFromDir deletes from a specific trash directory
func (m *Manager) DeleteFromDir(trashDir, trashName string) error {
	if err := validateFileName(trashName); err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}

	srcPath := FilesPath(trashDir, trashName)
	infoPath := TrashInfoPath(trashDir, trashName)

	if _, err := os.Lstat(srcPath); err != nil {
		return fmt.Errorf("file not in trash: %s", trashName)
	}

	if err := chmodAndRemoveAll(srcPath); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	os.Remove(infoPath)

	return nil
}

// FindByName finds items in trash matching a name (supports partial match)
func (m *Manager) FindByName(name string) ([]TrashItem, error) {
	items, err := m.List()
	if err != nil {
		return nil, err
	}

	var matches []TrashItem
	for _, item := range items {
		if item.Name == name || strings.HasPrefix(item.Name, name) {
			matches = append(matches, item)
		}
	}

	return matches, nil
}

// GetLast returns the most recently trashed item
func (m *Manager) GetLast() (*TrashItem, error) {
	items, err := m.List()
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("trash is empty")
	}

	return &items[0], nil
}

// Empty clears all items from the trash
func (m *Manager) Empty() error {
	return m.EmptyOlderThan(0)
}

// EmptyOlderThan clears items older than days days from all directories
func (m *Manager) EmptyOlderThan(days int) error {
	dirs, err := GetAllTrashDirs()
	if err != nil {
		return fmt.Errorf("get trash directories: %w", err)
	}

	for _, dir := range dirs {
		if err := m.EmptyDirOlderThan(dir, days); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) EmptyDirOlderThan(trashDir string, days int) error {
	items, err := m.ListFromDir(trashDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	if days == 0 {
		cutoff = time.Now()
	}

	for _, item := range items {
		if days == 0 || item.DeletionDate.Before(cutoff) {
			// Validate filename to prevent path traversal
			if err := validateFileName(item.Name); err != nil {
				return fmt.Errorf("invalid filename %s: %w", item.Name, err)
			}

			// Remove file/directory using chmodAndRemoveAll (handles read-only files)
			// Path is safe: trashDir is validated, item.Name is validated
			filePath := filepath.Join(trashDir, "files", item.Name)
			if err := chmodAndRemoveAll(filePath); err != nil {
				return fmt.Errorf("remove %s: %w", item.Name, err)
			}

			// Remove trashinfo
			infoPath := filepath.Join(trashDir, "info", item.Name+".trashinfo")
			os.Remove(infoPath) // Best effort
		}
	}

	return nil
}

// chmodAndRemoveAll makes all files writable then removes the directory tree
func chmodAndRemoveAll(path string) error {
	// First, make everything writable so we can delete it
	filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors, best effort
		}
		// Make files and directories writable
		if info.Mode()&0200 == 0 {
			os.Chmod(subPath, info.Mode()|0200)
		}
		return nil
	})

	return os.RemoveAll(path)
}

// Status returns statistics about the trash from all directories
func (m *Manager) Status() (count int, totalSize int64, err error) {
	dirs, err := GetAllTrashDirs()
	if err != nil {
		return 0, 0, fmt.Errorf("get trash directories: %w", err)
	}

	for _, dir := range dirs {
		c, s, err := m.StatusFromDir(dir)
		if err != nil {
			continue
		}
		count += c
		totalSize += s
	}

	return count, totalSize, nil
}

// StatusFromDir returns statistics from a specific trash directory
func (m *Manager) StatusFromDir(trashDir string) (count int, totalSize int64, err error) {
	items, err := m.ListFromDir(trashDir)
	if err != nil {
		return 0, 0, err
	}

	for _, item := range items {
		count++
		if item.IsDir {
			// Calculate directory size
			size, _ := dirSize(FilesPath(trashDir, item.Name))
			totalSize += size
		} else {
			totalSize += item.Size
		}
	}

	return count, totalSize, nil
}

// dirSize calculates the total size of a directory
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// GetLargest returns the N largest items in trash
func (m *Manager) GetLargest(n int) ([]TrashItem, error) {
	items, err := m.List()
	if err != nil {
		return nil, err
	}

	// Sort by size
	sort.Slice(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})

	if len(items) < n {
		return items, nil
	}
	return items[:n], nil
}

// GetOldestAndNewest returns the oldest and newest deletion dates
func (m *Manager) GetOldestAndNewest() (oldest, newest time.Time, err error) {
	items, err := m.List()
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if len(items) == 0 {
		return time.Time{}, time.Time{}, nil
	}

	// Items are already sorted newest first
	newest = items[0].DeletionDate
	oldest = items[len(items)-1].DeletionDate

	return oldest, newest, nil
}

// validateFileName checks for path traversal attempts in filenames
func validateFileName(name string) error {
	// Block empty names
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid filename: %q", name)
	}

	// Block path separators
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("filename contains path separator: %q", name)
	}

	// Block parent directory references
	if strings.Contains(name, "..") {
		return fmt.Errorf("filename contains path traversal: %q", name)
	}

	// Block null bytes
	if strings.ContainsRune(name, '\x00') {
		return fmt.Errorf("filename contains null byte")
	}

	return nil
}

// safeRemoveAll removes a path without following symlinks in subdirectories
func safeRemoveAll(path string) error {
	// Check if path itself is a symlink
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return err
	}

	// If it's a file or symlink, just remove it
	if !info.IsDir() {
		return os.Remove(path)
	}

	// For directories, walk and check for symlinks before removing
	err = filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check for symlinks inside directory
		if subPath != path && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to remove directory containing symlink: %s", subPath)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return os.RemoveAll(path)
}

// isCrossDeviceError checks if the error is a cross-device link error
func isCrossDeviceError(err error) bool {
	if err == nil {
		return false
	}
	// Use syscall.EXDEV for portable cross-device detection
	if linkErr, ok := err.(*os.LinkError); ok {
		return linkErr.Err == syscall.EXDEV
	}
	// Fallback for non-LinkError cases (shouldn't happen with os.Rename)
	return false
}
