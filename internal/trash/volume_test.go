package trash

import (
	"fmt"
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

func TestTrashVolumeName(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/home/user/.local/share")

	tests := []struct {
		name     string
		trashDir string
		expected string
	}{
		{name: "empty", trashDir: "", expected: "Unknown"},
		{name: "home trash", trashDir: "/home/user/.local/share/Trash", expected: "Home"},
		{name: "private volume trash", trashDir: "/mnt/usb/.Trash-1000", expected: "/mnt/usb"},
		{name: "shared volume trash", trashDir: "/media/disk/.Trash/1000", expected: "/media/disk"},
		{name: "root volume trash", trashDir: "/.Trash-1000", expected: "/"},
		{name: "unknown layout", trashDir: "/custom/trash", expected: "/custom/trash"},
		{name: "invalid private suffix", trashDir: "/mnt/usb/.Trash-user", expected: "/mnt/usb/.Trash-user"},
		{name: "invalid shared suffix", trashDir: "/mnt/usb/.Trash/user", expected: "/mnt/usb/.Trash/user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, TrashVolumeName(tt.trashDir))
		})
	}
}

func TestTrashItemVolumeName(t *testing.T) {
	item := TrashItem{TrashDir: "/mnt/usb/.Trash-1000"}

	assert.Equal(t, "/mnt/usb", item.VolumeName())
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

func TestGetFsType(t *testing.T) {
	tmpDir := t.TempDir()
	mountsFile := filepath.Join(tmpDir, "mounts")

	// Create a mock mounts file
	// We need to use a real mount point from the system for getMountPoint to work,
	// or we can mock the whole thing.
	// Let's use "/" which is always a mount point.
	rootMount, err := getMountPoint("/")
	require.NoError(t, err)

	err = os.WriteFile(mountsFile, []byte(fmt.Sprintf("/dev/root %s ext4 rw 0 0\n", rootMount)), 0644)
	require.NoError(t, err)

	SetMountsFilePath(mountsFile)
	defer SetMountsFilePath("/dev/null")

	fstype, err := getFsType("/")
	require.NoError(t, err)
	assert.Equal(t, "ext4", fstype)
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
