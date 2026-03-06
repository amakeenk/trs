package trash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeTrashDir(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	t.Run("with XDG_DATA_HOME", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("XDG_DATA_HOME", tmpDir)

		got, err := HomeTrashDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "Trash"), got)
	})

	t.Run("without XDG_DATA_HOME", func(t *testing.T) {
		os.Unsetenv("XDG_DATA_HOME")

		got, err := HomeTrashDir()
		require.NoError(t, err)

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".local", "share", "Trash")
		assert.Equal(t, expected, got)
	})
}

// TestGetMountPointWithNonExistentPath tests getMountPoint with deeply nested non-existent path
func TestGetMountPointWithNonExistentPath(t *testing.T) {
	// This should walk up to find an existing parent
	mount, err := getMountPoint("/nonexistent/deeply/nested/path/that/does/not/exist")
 require.NoError(t, err)
	// Should return a valid mount point (likely "/")
	assert.NotEmpty(t, mount)
}

// TestSameFilesystemWithNonExistent tests SameFilesystem with non-existent path
func TestSameFilesystemWithNonExistent(t *testing.T) {
	_, err := SameFilesystem("/nonexistent/path1", "/nonexistent/path2")
 require.Error(t, err)
}

// TestEnsureTrashDirWithSymlink tests EnsureTrashDir rejects symlinks
func TestEnsureTrashDirWithSymlink(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	require.NoError(t, os.MkdirAll(targetDir, 0700))

	symlinkDir := filepath.Join(tmpDir, "symlink_trash")
 require.NoError(t, os.Symlink(targetDir, symlinkDir))

	err := EnsureTrashDir(symlinkDir)
 require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}


func TestEnsureTrashDir(t *testing.T) {
	trashDir := t.TempDir() + "/Trash"

	err := EnsureTrashDir(trashDir)
	require.NoError(t, err)

	// Check all directories exist
	assert.DirExists(t, trashDir)
	assert.DirExists(t, filepath.Join(trashDir, "files"))
	assert.DirExists(t, filepath.Join(trashDir, "info"))
}

func TestEnsureTrashDirAlreadyExists(t *testing.T) {
	trashDir := t.TempDir() + "/Trash"

	// Create twice
	err := EnsureTrashDir(trashDir)
	require.NoError(t, err)

	err = EnsureTrashDir(trashDir)
	require.NoError(t, err)
}

func TestGetMountPoint(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file in temp dir
	file := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(file, []byte("test"), 0644)
	require.NoError(t, err)

	mount, err := getMountPoint(file)
	require.NoError(t, err)
	assert.NotEmpty(t, mount)

	// The mount point should be the same for the directory
	dirMount, err := getMountPoint(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, mount, dirMount)
}

func TestSameFilesystem(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("test"), 0644)
	require.NoError(t, err)

	same, err := SameFilesystem(file1, file2)
	require.NoError(t, err)
	assert.True(t, same)
}

func TestGetTrashDirForPath(t *testing.T) {
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(file, []byte("test"), 0644)
	require.NoError(t, err)

	trashDir, err := GetTrashDirForPath(file)
	require.NoError(t, err)
	assert.NotEmpty(t, trashDir)

	// Verify directories were created
	assert.DirExists(t, filepath.Join(trashDir, "files"))
	assert.DirExists(t, filepath.Join(trashDir, "info"))
}
