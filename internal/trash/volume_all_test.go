package trash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFromDir_PopulatesTrashDir(t *testing.T) {
	// Create a temporary trash directory structure
	tmpDir := t.TempDir()

	// Set up XDG_DATA_HOME to use our temp directory
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create home trash structure
	homeTrash := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(homeTrash, "files")
	infoDir := filepath.Join(homeTrash, "info")
	require.NoError(t, os.MkdirAll(filesDir, 0755))
	require.NoError(t, os.MkdirAll(infoDir, 0755))

	// Create a test file in trash
	testFile := filepath.Join(filesDir, "test_file.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	// Create trashinfo file
	trashInfo := `[Trash Info]
Path=/home/user/original/test_file.txt
DeletionDate=2026-03-05T10:00:00`
	require.NoError(t, os.WriteFile(filepath.Join(infoDir, "test_file.txt.trashinfo"), []byte(trashInfo), 0644))

	// Create manager
	mgr, err := NewManager()
	require.NoError(t, err)

	// List from this specific directory
	items, err := mgr.ListFromDir(homeTrash)
	require.NoError(t, err)

	// Verify results
	require.Len(t, items, 1, "Expected 1 item in list")
	assert.Equal(t, "test_file.txt", items[0].Name)
	assert.Equal(t, "/home/user/original/test_file.txt", items[0].OriginalPath)
	assert.Equal(t, homeTrash, items[0].TrashDir, "TrashDir should be populated")
}

func TestList_AggregatesFromMultipleDirs(t *testing.T) {
	// This test verifies that List() aggregates items from all trash directories
	// We can't easily mock GetAllTrashDirs() since it reads /proc/mounts,
	// but we can verify that the home trash is always included

	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create home trash structure
	homeTrash := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(homeTrash, "files")
	infoDir := filepath.Join(homeTrash, "info")
	require.NoError(t, os.MkdirAll(filesDir, 0755))
	require.NoError(t, os.MkdirAll(infoDir, 0755))

	// Create two test files in home trash
	for i := 1; i <= 2; i++ {
		testFile := filepath.Join(filesDir, filepath.FromSlash("test_file_"+string(rune('0'+i))+".txt"))
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		trashInfo := `[Trash Info]
Path=/home/user/file` + string(rune('0'+i)) + `.txt
DeletionDate=2026-03-05T10:00:00`
		require.NoError(t, os.WriteFile(filepath.Join(infoDir, "test_file_"+string(rune('0'+i))+".txt.trashinfo"), []byte(trashInfo), 0644))
	}

	// Create manager
	mgr, err := NewManager()
	require.NoError(t, err)

	// List all items
	items, err := mgr.List()
	require.NoError(t, err)

	// Should have at least the 2 items we created
	assert.GreaterOrEqual(t, len(items), 2, "List should return items from home trash")

	// All items should have TrashDir populated
	for _, item := range items {
		assert.NotEmpty(t, item.TrashDir, "TrashDir should be populated for all items")
	}
}

func TestStatus_AggregatesFromMultipleDirs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create home trash structure
	homeTrash := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(homeTrash, "files")
	infoDir := filepath.Join(homeTrash, "info")
	require.NoError(t, os.MkdirAll(filesDir, 0755))
	require.NoError(t, os.MkdirAll(infoDir, 0755))

	// Create test file
	testFile := filepath.Join(filesDir, "test_file.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content here"), 0644))

	trashInfo := `[Trash Info]
Path=/home/user/test_file.txt
DeletionDate=2026-03-05T10:00:00`
	require.NoError(t, os.WriteFile(filepath.Join(infoDir, "test_file.txt.trashinfo"), []byte(trashInfo), 0644))

	// Create manager
	mgr, err := NewManager()
	require.NoError(t, err)

	// Get status
	count, totalSize, err := mgr.Status()
	require.NoError(t, err)

	assert.GreaterOrEqual(t, count, 1, "Status should count items from home trash")
	assert.GreaterOrEqual(t, totalSize, int64(16), "Total size should include file size")
}

// TestGetAllTrashDirs_ReturnsHomeTrash verifies that GetAllTrashDirs always includes home trash
func TestGetAllTrashDirs_ReturnsHomeTrash(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create home trash structure so it exists
	homeTrash := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(filepath.Join(homeTrash, "files"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(homeTrash, "info"), 0755))

	dirs, err := GetAllTrashDirs()
	require.NoError(t, err)

	// Home trash should always be included
	assert.Contains(t, dirs, homeTrash, "Home trash should be in the list")
}

// TestCrossVolumeTrash tests trashing a file to a different filesystem
// This test works by trashing a file from /tmp and verifying it's visible
func TestCrossVolumeTrash(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create home trash with secure permissions (0700)
	homeTrash := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(filepath.Join(homeTrash, "files"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(homeTrash, "info"), 0700))

	mgr, err := NewManager()
	require.NoError(t, err)

	// Create a file directly in /tmp (on tmpfs)
	// Use a unique filename to avoid conflicts
	tmpFile := filepath.Join("/tmp", "trs_test_crossvol_"+filepath.Base(tmpDir)+".txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("cross-volume test"), 0644))
	defer os.Remove(tmpFile) // cleanup if test fails

	// Trash the file
	err = mgr.Move(tmpFile)
	require.NoError(t, err, "Move should succeed")

	// Verify the file is visible in list (regardless of which trash it went to)
	items, err := mgr.List()
	require.NoError(t, err)

	var found bool
	for _, item := range items {
		if item.OriginalPath == tmpFile {
			found = true
			// Verify TrashDir is populated
			assert.NotEmpty(t, item.TrashDir, "TrashDir should be set")
			break
		}
	}
	assert.True(t, found, "Trashed file should be visible in List()")

	// Verify status includes the file
	count, _, err := mgr.Status()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "Status should count the file")

	// Verify we can restore the file
	err = mgr.Restore(filepath.Base(tmpFile), false)
	require.NoError(t, err, "Restore should succeed")

	// Verify file was restored
	_, err = os.Stat(tmpFile)
	assert.NoError(t, err, "File should exist after restore")
	os.Remove(tmpFile) // final cleanup
}

// TestRestoreFromVolumeTrash tests restoring a file that was trashed to volume trash
func TestRestoreFromVolumeTrash(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create home trash with secure permissions
	homeTrash := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(filepath.Join(homeTrash, "files"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(homeTrash, "info"), 0700))

	mgr, err := NewManager()
	require.NoError(t, err)

	// Create a test file in /tmp
	tmpFile := filepath.Join("/tmp", "trs_restore_test_"+filepath.Base(tmpDir)+".txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("restore test"), 0644))
	defer os.Remove(tmpFile) // cleanup

	// Trash the file
	require.NoError(t, mgr.Move(tmpFile), "Move should succeed")

	// Restore the file
	err = mgr.Restore(filepath.Base(tmpFile), false)
	require.NoError(t, err, "Restore should succeed")

	// Verify file was restored to original location
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err, "File should exist at original location")
	assert.Equal(t, "restore test", string(content))
	os.Remove(tmpFile)
}
