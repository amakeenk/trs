package trash

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenTrashRoot(t *testing.T) {
	t.Run("successful open", func(t *testing.T) {
		tmpDir := t.TempDir()
		trashDir := filepath.Join(tmpDir, "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0700))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

		root, err := OpenTrashRoot(trashDir)
		require.NoError(t, err)
		require.NotNil(t, root)
		defer root.Close()
	})

	t.Run("non-existent directory", func(t *testing.T) {
		root, err := OpenTrashRoot("/nonexistent/trash/dir")
		require.Error(t, err)
		assert.Nil(t, root)
		assert.Contains(t, err.Error(), "open trash root")
	})
}

func TestTrashRootClose(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(trashDir, 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	require.NoError(t, root.Close())
}

func TestTrashRootPaths(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	assert.Equal(t, "files/test.txt", root.FilesPath("test.txt"))
	assert.Equal(t, "info/test.txt.trashinfo", root.InfoPath("test.txt"))
}

func TestTrashRootLstat(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(infoDir, 0700))

	// Create a test file
	testFile := filepath.Join(filesDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0600))

	// Create a test trashinfo
	ti := &TrashInfo{Path: "/original/path", DeletionDate: time.Now()}
	require.NoError(t, ti.Write(filepath.Join(infoDir, "test.txt.trashinfo")))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	t.Run("LstatFiles existing", func(t *testing.T) {
		info, err := root.LstatFiles("test.txt")
		require.NoError(t, err)
		assert.Equal(t, "test.txt", info.Name())
		assert.False(t, info.IsDir())
	})

	t.Run("LstatFiles non-existent", func(t *testing.T) {
		_, err := root.LstatFiles("nonexistent.txt")
		require.Error(t, err)
	})

	t.Run("LstatInfo existing", func(t *testing.T) {
		info, err := root.LstatInfo("test.txt")
		require.NoError(t, err)
		assert.False(t, info.IsDir())
	})

	t.Run("LstatInfo non-existent", func(t *testing.T) {
		_, err := root.LstatInfo("nonexistent.txt")
		require.Error(t, err)
	})
}

func TestTrashRootOpen(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(infoDir, 0700))

	// Create test files
	testFile := filepath.Join(filesDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0600))

	ti := &TrashInfo{Path: "/original/path", DeletionDate: time.Now()}
	infoFile := filepath.Join(infoDir, "test.txt.trashinfo")
	require.NoError(t, ti.Write(infoFile))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	t.Run("OpenFiles existing", func(t *testing.T) {
		f, err := root.OpenFiles("test.txt")
		require.NoError(t, err)
		f.Close()
	})

	t.Run("OpenFiles non-existent", func(t *testing.T) {
		_, err := root.OpenFiles("nonexistent.txt")
		require.Error(t, err)
	})

	t.Run("OpenInfo existing", func(t *testing.T) {
		f, err := root.OpenInfo("test.txt")
		require.NoError(t, err)
		f.Close()
	})

	t.Run("OpenInfo non-existent", func(t *testing.T) {
		_, err := root.OpenInfo("nonexistent.txt")
		require.Error(t, err)
	})
}

func TestTrashRootCreate(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	t.Run("CreateFiles", func(t *testing.T) {
		f, err := root.CreateFiles("newfile.txt")
		require.NoError(t, err)
		f.Close()

		// Verify file exists
		_, err = root.LstatFiles("newfile.txt")
		require.NoError(t, err)
	})

	t.Run("CreateInfo", func(t *testing.T) {
		f, err := root.CreateInfo("newfile.txt")
		require.NoError(t, err)
		f.Close()

		// Verify file exists
		_, err = root.LstatInfo("newfile.txt")
		require.NoError(t, err)
	})
}

func TestTrashRootRemove(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(infoDir, 0700))

	// Create test files
	testFile := filepath.Join(filesDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0600))

	ti := &TrashInfo{Path: "/original/path", DeletionDate: time.Now()}
	infoFile := filepath.Join(infoDir, "test.txt.trashinfo")
	require.NoError(t, ti.Write(infoFile))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	t.Run("RemoveFiles", func(t *testing.T) {
		err := root.RemoveFiles("test.txt")
		require.NoError(t, err)

		_, err = root.LstatFiles("test.txt")
		require.Error(t, err)
	})

	t.Run("RemoveInfo", func(t *testing.T) {
		err := root.RemoveInfo("test.txt")
		require.NoError(t, err)

		_, err = root.LstatInfo("test.txt")
		require.Error(t, err)
	})
}

func TestTrashRootMkdir(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(trashDir, 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	t.Run("MkdirFiles", func(t *testing.T) {
		err := root.MkdirFiles()
		require.NoError(t, err)

		info, err := root.root.Lstat("files")
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("MkdirInfo", func(t *testing.T) {
		err := root.MkdirInfo()
		require.NoError(t, err)

		info, err := root.root.Lstat("info")
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestTrashRootReadDirFiles(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "file1.txt"), []byte("1"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "file2.txt"), []byte("2"), 0600))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	entries, err := root.ReadDirFiles()
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify names
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}
	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
}

func TestTrashRootReadDirFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	entries, err := root.ReadDirFiles()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestParseTrashInfoFromRoot(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	infoDir := filepath.Join(trashDir, "info")
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0700))
	require.NoError(t, os.MkdirAll(infoDir, 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	t.Run("valid trashinfo", func(t *testing.T) {
		ti := &TrashInfo{
			Path:         "/home/user/file.txt",
			DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		}
		require.NoError(t, ti.Write(filepath.Join(infoDir, "valid.trashinfo")))

		got, err := ParseTrashInfoFromRoot(root, "valid")
		require.NoError(t, err)
		assert.Equal(t, ti.Path, got.Path)
		assert.Equal(t, ti.DeletionDate, got.DeletionDate)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := ParseTrashInfoFromRoot(root, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stat trashinfo")
	})

	t.Run("file too large", func(t *testing.T) {
		// Create a file larger than maxTrashInfoSize
		largeContent := "[Trash Info]\nPath=" + string(make([]byte, 9000)) + "\nDeletionDate=2024-01-15T10:30:00"
		require.NoError(t, os.WriteFile(filepath.Join(infoDir, "large.trashinfo"), []byte(largeContent), 0600))

		_, err := ParseTrashInfoFromRoot(root, "large")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file too large")
	})
}

func TestRemoveAll(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	t.Run("remove single file", func(t *testing.T) {
		// Create file
		f, err := root.CreateFiles("single.txt")
		require.NoError(t, err)
		f.Close()

		// Remove it
		err = root.RemoveAllFiles("single.txt")
		require.NoError(t, err)

		// Verify gone
		_, err = root.LstatFiles("single.txt")
		require.Error(t, err)
	})

	t.Run("remove empty directory", func(t *testing.T) {
		// Create directory
		require.NoError(t, root.root.Mkdir("files/emptydir", 0700))

		// Remove it
		err := root.RemoveAllFiles("emptydir")
		require.NoError(t, err)

		// Verify gone
		_, err = root.LstatFiles("emptydir")
		require.Error(t, err)
	})

	t.Run("remove directory with files", func(t *testing.T) {
		// Create directory with files
		require.NoError(t, root.root.Mkdir("files/withfiles", 0700))
		f, err := root.root.Create("files/withfiles/file1.txt")
		require.NoError(t, err)
		f.Close()
		f, err = root.root.Create("files/withfiles/file2.txt")
		require.NoError(t, err)
		f.Close()

		// Remove it
		err = root.RemoveAllFiles("withfiles")
		require.NoError(t, err)

		// Verify gone
		_, err = root.LstatFiles("withfiles")
		require.Error(t, err)
	})

	t.Run("remove nested directories", func(t *testing.T) {
		// Create nested structure
		require.NoError(t, root.root.Mkdir("files/nested", 0700))
		require.NoError(t, root.root.Mkdir("files/nested/sub1", 0700))
		require.NoError(t, root.root.Mkdir("files/nested/sub1/sub2", 0700))
		f, err := root.root.Create("files/nested/sub1/sub2/deep.txt")
		require.NoError(t, err)
		f.Close()
		f, err = root.root.Create("files/nested/top.txt")
		require.NoError(t, err)
		f.Close()

		// Remove it
		err = root.RemoveAllFiles("nested")
		require.NoError(t, err)

		// Verify gone
		_, err = root.LstatFiles("nested")
		require.Error(t, err)
	})

	t.Run("remove non-existent returns nil", func(t *testing.T) {
		err := root.RemoveAllFiles("nonexistent")
		require.NoError(t, err)
	})

	t.Run("remove directory containing symlink fails", func(t *testing.T) {
		// Create directory with a symlink inside
		require.NoError(t, root.root.Mkdir("files/withsymlink", 0700))
		f, err := root.root.Create("files/withsymlink/target.txt")
		require.NoError(t, err)
		f.Close()

		// Create symlink using os.Symlink (relative to files directory)
		targetPath := filepath.Join(filesDir, "target.txt")
		symlinkPath := filepath.Join(filesDir, "withsymlink", "link")
		require.NoError(t, os.Symlink(targetPath, symlinkPath))

		// Remove should fail due to symlink
		err = root.RemoveAllFiles("withsymlink")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "symlink")
	})
}

func TestRemoveAllFilesSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	// Create a file and a symlink to it
	f, err := root.CreateFiles("target.txt")
	require.NoError(t, err)
	f.Close()

	// Create symlink (the symlink itself should be removable)
	require.NoError(t, os.Symlink(filepath.Join(filesDir, "target.txt"), filepath.Join(filesDir, "symlink")))

	// Remove the symlink (not the target)
	err = root.RemoveAllFiles("symlink")
	require.NoError(t, err)

	// Verify symlink is gone
	_, err = root.LstatFiles("symlink")
	require.Error(t, err)

	// Verify target still exists
	_, err = root.LstatFiles("target.txt")
	require.NoError(t, err)
}

func TestRemoveAllRecursiveError(t *testing.T) {
	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	// Create nested structure with a symlink at the deepest level
	require.NoError(t, root.root.Mkdir("files/nested", 0700))
	require.NoError(t, root.root.Mkdir("files/nested/sub", 0700))

	// Create symlink inside nested directory
	require.NoError(t, os.Symlink("/some/target", filepath.Join(filesDir, "nested", "sub", "link")))

	// Remove should fail when encountering symlink in nested directory
	err = root.RemoveAllFiles("nested")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

func TestRemoveAllFilesError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	// Create a file
	f, err := root.CreateFiles("readonly.txt")
	require.NoError(t, err)
	f.Close()

	// Make the files directory non-writable to cause remove error
	require.NoError(t, os.Chmod(filesDir, 0500))
	defer os.Chmod(filesDir, 0700)

	// Remove should fail due to permission
	err = root.RemoveAllFiles("readonly.txt")
	require.Error(t, err)
}

func TestRemoveAllDirectoryRemoveError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	trashDir := filepath.Join(tmpDir, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	require.NoError(t, os.MkdirAll(filesDir, 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0700))

	root, err := OpenTrashRoot(trashDir)
	require.NoError(t, err)
	defer root.Close()

	// Create a directory with a file inside
	require.NoError(t, root.root.Mkdir("files/readonlydir", 0700))
	f, err := root.root.Create("files/readonlydir/file.txt")
	require.NoError(t, err)
	f.Close()

	// Make the parent directory non-writable to cause remove error
	require.NoError(t, os.Chmod(filesDir, 0500))
	defer os.Chmod(filesDir, 0700)

	// Remove should fail due to permission when trying to remove directory
	err = root.RemoveAllFiles("readonlydir")
	require.Error(t, err)
}
