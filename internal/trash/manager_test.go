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

func TestManager_FindByName(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Empty trash
	matches, err := mgr.FindByName("file")
	require.NoError(t, err)
	assert.Empty(t, matches)

	// Add files
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644))
	require.NoError(t, mgr.Move(filepath.Join(tmpDir, "file1.txt")))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644))
	require.NoError(t, mgr.Move(filepath.Join(tmpDir, "file2.txt")))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("3"), 0644))
	require.NoError(t, mgr.Move(filepath.Join(tmpDir, "other.txt")))

	// Exact match
	matches, err = mgr.FindByName("file1.txt")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "file1.txt", matches[0].Name)

	// Partial match
	matches, err = mgr.FindByName("file")
	require.NoError(t, err)
	require.Len(t, matches, 2)

	// No match
	matches, err = mgr.FindByName("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestManager_GetLargest(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Empty trash
	largest, err := mgr.GetLargest(3)
	require.NoError(t, err)
	assert.Empty(t, largest)

	// Add files with different sizes
	tmpDir := t.TempDir()
	small := filepath.Join(tmpDir, "small.txt")
	require.NoError(t, os.WriteFile(small, []byte("x"), 0644))
	require.NoError(t, mgr.Move(small))

	medium := filepath.Join(tmpDir, "medium.txt")
	require.NoError(t, os.WriteFile(medium, []byte("xxxxx"), 0644))
	require.NoError(t, mgr.Move(medium))

	big := filepath.Join(tmpDir, "big.txt")
	require.NoError(t, os.WriteFile(big, []byte("xxxxxxxxxxx"), 0644))
	require.NoError(t, mgr.Move(big))

	// Get top 2
	largest, err = mgr.GetLargest(2)
	require.NoError(t, err)
	require.Len(t, largest, 2)
	assert.Equal(t, "big.txt", largest[0].Name)
	assert.Equal(t, "medium.txt", largest[1].Name)

	// Get more than available
	largest, err = mgr.GetLargest(10)
	require.NoError(t, err)
	require.Len(t, largest, 3)
}

func TestManager_GetOldestAndNewest(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Empty trash
	oldest, newest, err := mgr.GetOldestAndNewest()
	require.NoError(t, err)
	assert.True(t, oldest.IsZero())
	assert.True(t, newest.IsZero())

	// Add files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "first.txt")
	require.NoError(t, os.WriteFile(file1, []byte("1"), 0644))
	require.NoError(t, mgr.Move(file1))

	time.Sleep(time.Second)

	file2 := filepath.Join(tmpDir, "second.txt")
	require.NoError(t, os.WriteFile(file2, []byte("2"), 0644))
	require.NoError(t, mgr.Move(file2))

	oldest, newest, err = mgr.GetOldestAndNewest()
	require.NoError(t, err)
	assert.False(t, oldest.IsZero())
	assert.False(t, newest.IsZero())
	assert.True(t, newest.After(oldest))
}

func TestDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory
	size, err := dirSize(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// Add files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("world"), 0644))

	size, err = dirSize(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, int64(10), size) // "hello" + "world" = 10 bytes

	// Add nested directory
	subDir := filepath.Join(tmpDir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644))

	size, err = dirSize(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, int64(16), size) // 10 + "nested" = 16 bytes
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello world"), 0644))

	dst := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(src, dst)
	require.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestCopyAndDelete(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	require.NoError(t, os.WriteFile(src, []byte("test content"), 0644))

	dst := filepath.Join(tmpDir, "dest.txt")

	mgr, err := NewManager()
	require.NoError(t, err)

	err = mgr.copyAndDelete(src, dst)
	require.NoError(t, err)

	// Verify source is gone
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))

	// Verify content at destination
	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestCopyDirAndDelete(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "sourcedir")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("content2"), 0644))

	// Create symlink
	require.NoError(t, os.Symlink(filepath.Join(srcDir, "file1.txt"), filepath.Join(srcDir, "link.txt")))

	dstDir := filepath.Join(tmpDir, "destdir")

	mgr, err := NewManager()
	require.NoError(t, err)

	err = mgr.copyDirAndDelete(srcDir, dstDir)
	require.NoError(t, err)

	// Verify source is gone
	_, err = os.Stat(srcDir)
	assert.True(t, os.IsNotExist(err))

	// Verify content at destination
	assert.FileExists(t, filepath.Join(dstDir, "file1.txt"))
	assert.FileExists(t, filepath.Join(dstDir, "sub", "file2.txt"))

	// Verify symlink preserved
	linkInfo, err := os.Lstat(filepath.Join(dstDir, "link.txt"))
	require.NoError(t, err)
	assert.True(t, linkInfo.Mode()&os.ModeSymlink != 0)
}

func TestManager_StatusWithDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	// Add a directory with files
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdir")
	require.NoError(t, os.MkdirAll(testDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("world"), 0644))
	require.NoError(t, mgr.Move(testDir))

	// Status should include directory size
	count, size, err := mgr.Status()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, int64(10), size) // "hello" + "world" = 10 bytes
}
