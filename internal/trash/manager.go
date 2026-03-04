package trash

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TrashItem represents a file in the trash
type TrashItem struct {
	Name         string    // Name in trash
	OriginalPath string    // Original absolute path
	DeletionDate time.Time // When it was trashed
	Size         int64     // Size in bytes
	IsDir        bool      // Is directory
}

// Manager handles trash operations
type Manager struct {
	homeTrash string
}

// NewManager creates a new trash manager
func NewManager() (*Manager, error) {
	homeTrash, err := HomeTrashDir()
	if err != nil {
		return nil, err
	}

	return &Manager{homeTrash: homeTrash}, nil
}

// Move moves a file or directory to the trash
func (m *Manager) Move(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	// Check if path exists (use Lstat for symlinks)
	_, err = os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", absPath)
		}
		return fmt.Errorf("stat %s: %w", absPath, err)
	}

	// Get appropriate trash directory
	trashDir, err := GetTrashDirForPath(absPath)
	if err != nil {
		return fmt.Errorf("get trash directory: %w", err)
	}

	// Resolve name conflicts
	trashName, err := m.resolveNameConflict(trashDir, filepath.Base(absPath))
	if err != nil {
		return fmt.Errorf("resolve name conflict: %w", err)
	}

	// Create trashinfo
	ti := &TrashInfo{
		Path:         absPath,
		DeletionDate: time.Now(),
	}

	// Write trashinfo first
	infoPath := TrashInfoPath(trashDir, trashName)
	if err := ti.Write(infoPath); err != nil {
		return fmt.Errorf("write trashinfo: %w", err)
	}

	// Move the file/directory
	destPath := FilesPath(trashDir, trashName)
	if err := m.movePath(absPath, destPath); err != nil {
		// Clean up trashinfo on failure
		os.Remove(infoPath)
		return fmt.Errorf("move to trash: %w", err)
	}

	return nil
}

// movePath handles moving a file/directory, including cross-device moves
func (m *Manager) movePath(src, dst string) error {
	// Try rename first (fast, atomic)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If cross-device, fall back to copy + delete
	if strings.Contains(err.Error(), "invalid cross-device link") {
		return m.copyAndDelete(src, dst)
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

	// Copy file
	if err := copyFile(src, dst); err != nil {
		return err
	}

	// Delete original
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
		return err
	}

	// Delete original directory
	return os.RemoveAll(src)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// resolveNameConflict adds suffix _1, _2, etc. if name already exists in trash
func (m *Manager) resolveNameConflict(trashDir, name string) (string, error) {
	trashName := name
	counter := 0

	for {
		filesPath := FilesPath(trashDir, trashName)
		infoPath := TrashInfoPath(trashDir, trashName)

		_, filesErr := os.Stat(filesPath)
		_, infoErr := os.Stat(infoPath)

		if os.IsNotExist(filesErr) && os.IsNotExist(infoErr) {
			return trashName, nil
		}

		counter++
		trashName = fmt.Sprintf("%s_%d", name, counter)
	}
}

// List returns all items in the trash
func (m *Manager) List() ([]TrashItem, error) {
	return m.ListFromDir(m.homeTrash)
}

// ListFromDir returns items from a specific trash directory
func (m *Manager) ListFromDir(trashDir string) ([]TrashItem, error) {
	filesDir := filepath.Join(trashDir, "files")

	entries, err := os.ReadDir(filesDir)
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

		items = append(items, TrashItem{
			Name:         entry.Name(),
			OriginalPath: ti.Path,
			DeletionDate: ti.DeletionDate,
			Size:         info.Size(),
			IsDir:        entry.IsDir(),
		})
	}

	// Sort by deletion date, newest first
	sort.Slice(items, func(i, j int) bool {
		return items[i].DeletionDate.After(items[j].DeletionDate)
	})

	return items, nil
}

// Restore restores a file from the trash
func (m *Manager) Restore(trashName string, overwrite bool) error {
	return m.RestoreFromDir(m.homeTrash, trashName, overwrite)
}

// RestoreFromDir restores from a specific trash directory
func (m *Manager) RestoreFromDir(trashDir, trashName string, overwrite bool) error {
	infoPath := TrashInfoPath(trashDir, trashName)
	ti, err := ParseTrashInfo(infoPath)
	if err != nil {
		return fmt.Errorf("read trashinfo: %w", err)
	}

	srcPath := FilesPath(trashDir, trashName)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("file not in trash: %s", trashName)
	}

	// Check if destination exists
	if _, err := os.Stat(ti.Path); err == nil {
		if !overwrite {
			return fmt.Errorf("destination exists: %s (use --force to overwrite)", ti.Path)
		}
		// Remove existing destination
		if err := os.RemoveAll(ti.Path); err != nil {
			return fmt.Errorf("remove existing: %w", err)
		}
	}

	// Ensure destination directory exists
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

// EmptyOlderThan clears items older than days days
func (m *Manager) EmptyOlderThan(days int) error {
	return m.EmptyDirOlderThan(m.homeTrash, days)
}

// EmptyDirOlderThan clears items from a specific trash directory
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
			// Delete file
			filesPath := FilesPath(trashDir, item.Name)
			if err := os.RemoveAll(filesPath); err != nil {
				return fmt.Errorf("remove %s: %w", item.Name, err)
			}

			// Delete trashinfo
			infoPath := TrashInfoPath(trashDir, item.Name)
			os.Remove(infoPath)
		}
	}

	return nil
}

// Status returns statistics about the trash
func (m *Manager) Status() (count int, totalSize int64, err error) {
	return m.StatusFromDir(m.homeTrash)
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
