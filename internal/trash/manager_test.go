package trash

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_Move(t *testing.T) {
	// Set up temp home for XDG
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Create test file
	testFile := filepath.Join(t.TempDir(), "test.txt")
	err = os.WriteFile(testFile, []byte("hello"), 0644)
	require.NoError(t, err)

	// Move to trash
	err = mgr.Move(testFile)
	require.NoError(t, err)

	// Verify original is gone
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))

	// Verify file is in trash
	trashPath := filepath.Join(xdgData, "Trash", "files", "test.txt")
	_, err = os.Stat(trashPath)
	require.NoError(t, err)

	// Verify trashinfo exists
	infoPath := filepath.Join(xdgData, "Trash", "info", "test.txt.trashinfo")
	_, err = os.Stat(infoPath)
	require.NoError(t, err)
}

func TestManager_MoveDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Create test directory
	testDir := filepath.Join(t.TempDir(), "testdir")
	require.NoError(t, os.MkdirAll(testDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("content"), 0644))

	// Move to trash
	err = mgr.Move(testDir)
	require.NoError(t, err)

	// Verify directory is in trash
	trashPath := filepath.Join(xdgData, "Trash", "files", "testdir")
	assert.DirExists(t, trashPath)
	assert.FileExists(t, filepath.Join(trashPath, "file.txt"))
}

func TestManager_MoveSymlink(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Create target and symlink
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("target"), 0644))

	symlink := filepath.Join(tmpDir, "link.txt")
	require.NoError(t, os.Symlink(target, symlink))

	// Move symlink to trash
	err = mgr.Move(symlink)
	require.NoError(t, err)

	// Verify symlink is gone
	_, err = os.Lstat(symlink)
	assert.True(t, os.IsNotExist(err))

	// Verify target still exists
	assert.FileExists(t, target)

	// Verify symlink is in trash
	trashPath := filepath.Join(xdgData, "Trash", "files", "link.txt")
	info, err := os.Lstat(trashPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)
}

func TestManager_MoveNameConflict(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()

	// Create and move first file
	file1 := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file1, []byte("first"), 0644))
	require.NoError(t, mgr.Move(file1))

	// Create and move second file with same name
	file2 := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file2, []byte("second"), 0644))
	require.NoError(t, mgr.Move(file2))

	// Verify both exist in trash
	assert.FileExists(t, filepath.Join(xdgData, "Trash", "files", "test.txt"))
	assert.FileExists(t, filepath.Join(xdgData, "Trash", "files", "test.txt_1"))
}

func TestManager_List(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Empty trash
	items, err := mgr.List()
	require.NoError(t, err)
	assert.Empty(t, items)

	// Add files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	require.NoError(t, os.WriteFile(file1, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file1))

	time.Sleep(time.Second) // Ensure different timestamps

	file2 := filepath.Join(tmpDir, "file2.txt")
	require.NoError(t, os.WriteFile(file2, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file2))

	// List
	items, err = mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Verify sorted newest first
	assert.Equal(t, "file2.txt", items[0].Name)
	assert.Equal(t, "file1.txt", items[1].Name)
}

func TestManager_Restore(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Create and trash file
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(originalPath, []byte("hello"), 0644))
	require.NoError(t, mgr.Move(originalPath))

	// Restore
	err = mgr.Restore("test.txt", false)
	require.NoError(t, err)

	// Verify file is back
	content, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	// Verify removed from trash
	items, _ := mgr.List()
	assert.Empty(t, items)
}

func TestManager_RestoreConflict(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Create and trash file
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(originalPath, []byte("original"), 0644))
	require.NoError(t, mgr.Move(originalPath))

	// Create new file at same location
	require.NoError(t, os.WriteFile(originalPath, []byte("new"), 0644))

	// Restore without overwrite should fail
	err = mgr.Restore("test.txt", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination exists")

	// Restore with overwrite
	err = mgr.Restore("test.txt", true)
	require.NoError(t, err)

	// Verify original content
	content, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	assert.Equal(t, "original", string(content))
}

func TestManager_Empty(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Add files
	tmpDir := t.TempDir()
	for i := 0; i < 3; i++ {
		file := filepath.Join(tmpDir, "file.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))
	}

	// Verify files in trash
	items, _ := mgr.List()
	require.Len(t, items, 3)

	// Empty trash
	require.NoError(t, mgr.Empty())

	// Verify empty
	items, _ = mgr.List()
	assert.Empty(t, items)
}

func TestManager_Status(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Empty trash
	count, size, err := mgr.Status()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, int64(0), size)

	// Add file
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello world"), 0644))
	require.NoError(t, mgr.Move(file))

	// Check status
	count, size, err = mgr.Status()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Greater(t, size, int64(0))
}

func TestManager_GetLast(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Empty trash
	_, err = mgr.GetLast()
	assert.Error(t, err)

	// Add files
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "first.txt"), []byte("1"), 0644))
	require.NoError(t, mgr.Move(filepath.Join(tmpDir, "first.txt")))

	time.Sleep(time.Second)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "second.txt"), []byte("2"), 0644))
	require.NoError(t, mgr.Move(filepath.Join(tmpDir, "second.txt")))

	// Get last
	last, err := mgr.GetLast()
	require.NoError(t, err)
	assert.Equal(t, "second.txt", last.Name)
}

func TestResolveNameConflict(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	trashDir := filepath.Join(xdgData, "Trash")
	require.NoError(t, EnsureTrashDir(trashDir))

	// No conflict
	name, err := mgr.resolveNameConflict(trashDir, "file.txt")
	require.NoError(t, err)
	assert.Equal(t, "file.txt", name)

	// Create existing file
	os.WriteFile(FilesPath(trashDir, "file.txt"), []byte("test"), 0644)

	name, err = mgr.resolveNameConflict(trashDir, "file.txt")
	require.NoError(t, err)
	assert.Equal(t, "file.txt_1", name)

	// Create more conflicts
	os.WriteFile(FilesPath(trashDir, "file.txt_1"), []byte("test"), 0644)

	name, err = mgr.resolveNameConflict(trashDir, "file.txt")
	require.NoError(t, err)
	assert.Equal(t, "file.txt_2", name)
}
