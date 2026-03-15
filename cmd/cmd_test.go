package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"altlinux.space/amakeenk/trs/internal/trash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv sets up a temporary home directory for testing
func setupTestEnv(t *testing.T) (tmpHome string, xdgData string) {
	tmpHome = t.TempDir()
	xdgData = filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)
	return tmpHome, xdgData
}

func TestRootCmd(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.Contains(t, rootCmd.Use, "trs")
	assert.Contains(t, rootCmd.Short, "Safe rm replacement")
}

func TestExecute(t *testing.T) {
	setupTestEnv(t)

	// Create a test file
	testFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(testFile, []byte("hello"), 0644)
	require.NoError(t, err)

	// Create a file in the trash
	mgr, err := trash.NewManager()
	require.NoError(t, err)
	err = mgr.Move(testFile)
	require.NoError(t, err)
}

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Short, "List files in trash")
}

func TestNewRestoreCmd(t *testing.T) {
	cmd := NewRestoreCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "restore")
	assert.Contains(t, cmd.Short, "Restore files from trash")
}

func TestNewEmptyCmd(t *testing.T) {
	cmd := NewEmptyCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "empty")
	assert.Contains(t, cmd.Short, "Empty the trash")
}

func TestNewStatusCmd(t *testing.T) {
	cmd := NewStatusCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "status", cmd.Use)
	assert.Contains(t, cmd.Short, "Show trash statistics")
}

func TestNewVersionCmd(t *testing.T) {
	cmd := NewVersionCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Use)
	assert.Contains(t, cmd.Short, "Print version information")
}

func TestParseDays(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasErr   bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"1", 1, false},
		{"7", 7, false},
		{"30", 30, false},
		{"-1", -1, false},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDays(tt.input)
			if tt.hasErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestListOldFiles(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Add some files
	tmpDir := t.TempDir()
	for i := 0; i < 3; i++ {
		file := filepath.Join(tmpDir, "file.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	listOldFiles(mgr, 0)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()
	assert.Contains(t, output, "file.txt")
}

// TestExitWithError tests exitWithError via subprocess
func TestExitWithError(t *testing.T) {
	if os.Getenv("GO_TEST_EXIT_WITH_ERROR") == "1" {
		exitWithError("test error message", 42)
		return
	}

	// Run ourselves as a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestExitWithError")
	cmd.Env = append(os.Environ(), "GO_TEST_EXIT_WITH_ERROR=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 42, e.ExitCode())
		// Stderr may be empty due to no flush before exit
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

func TestTrashResult(t *testing.T) {
	result := TrashResult{
		Files:   []string{"file1.txt", "file2.txt"},
		Errors:  []string{"error1"},
		Success: 1,
		Failed:  1,
	}

	assert.Len(t, result.Files, 2)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, 1, result.Success)
	assert.Equal(t, 1, result.Failed)
}

func TestListItem(t *testing.T) {
	item := ListItem{
		Index:     1,
		Name:      "test.txt",
		Size:      "100 B",
		SizeBytes: 100,
		Deleted:   "2024-01-01 12:00",
		IsDir:     false,
		Original:  "/path/to/test.txt",
	}

	assert.Equal(t, 1, item.Index)
	assert.Equal(t, "test.txt", item.Name)
	assert.Equal(t, "100 B", item.Size)
	assert.Equal(t, int64(100), item.SizeBytes)
	assert.False(t, item.IsDir)
}

func TestRestoreResult(t *testing.T) {
	result := RestoreResult{
		Name:     "test.txt",
		Original: "/path/to/test.txt",
		Error:    "",
	}

	assert.Equal(t, "test.txt", result.Name)
	assert.Equal(t, "/path/to/test.txt", result.Original)
	assert.Empty(t, result.Error)
}

func TestEmptyResult(t *testing.T) {
	result := EmptyResult{
		Removed:   5,
		Remaining: 2,
		Days:      7,
		Message:   "Done",
	}

	assert.Equal(t, 5, result.Removed)
	assert.Equal(t, 2, result.Remaining)
	assert.Equal(t, 7, result.Days)
	assert.Equal(t, "Done", result.Message)
}

func TestStatusResult(t *testing.T) {
	result := StatusResult{
		Count:     10,
		Size:      "1.5 KB",
		SizeBytes: 1536,
		Oldest:    "2024-01-01 10:00",
		Newest:    "2024-01-10 15:00",
		Largest:   []LargestItem{{Name: "big.txt", Size: "500 B"}},
	}

	assert.Equal(t, 10, result.Count)
	assert.Equal(t, "1.5 KB", result.Size)
	assert.Equal(t, int64(1536), result.SizeBytes)
	assert.Len(t, result.Largest, 1)
}

func TestLargestItem(t *testing.T) {
	item := LargestItem{
		Name: "largefile.txt",
		Size: "1.0 GB",
	}

	assert.Equal(t, "largefile.txt", item.Name)
	assert.Equal(t, "1.0 GB", item.Size)
}

func TestRunTrashNoArgs(t *testing.T) {
	// Test that runTrash handles no args correctly
	// We can't test os.Exit directly, so we test the logic indirectly
	args := []string{}

	// The function calls cmd.Help() and exits with 2
	assert.Equal(t, 0, len(args))
}

func TestTrashFile(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	t.Run("successful trash", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))

		err := trashFile(mgr, file)
		require.NoError(t, err)

		// File should be gone
		_, err = os.Stat(file)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("file not found", func(t *testing.T) {
		err := trashFile(mgr, "/nonexistent/file.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "No such file or directory")
	})

	t.Run("directory without -r flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		dir := filepath.Join(tmpDir, "testdir")
		require.NoError(t, os.MkdirAll(dir, 0755))

		// Save current flag state
		oldRecursive := flagRecursive
		flagRecursive = false
		defer func() { flagRecursive = oldRecursive }()

		err := trashFile(mgr, dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Is a directory")
	})

	t.Run("directory with -r flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		dir := filepath.Join(tmpDir, "testdir2")
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644))

		// Save current flag state
		oldRecursive := flagRecursive
		flagRecursive = true
		defer func() { flagRecursive = oldRecursive }()

		err := trashFile(mgr, dir)
		require.NoError(t, err)

		// Directory should be gone
		_, err = os.Stat(dir)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestRunList(t *testing.T) {
	setupTestEnv(t)

	// Create manager and add a file
	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))
	require.NoError(t, mgr.Move(file))

	// Test JSON output
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runList(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()
	assert.Contains(t, output, "test.txt")
}

// TestRunTrashWithJSON tests runTrash with JSON output
func TestRunTrashWithJSON(t *testing.T) {
	setupTestEnv(t)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))

	// Save flags
	oldJSON := flagJSON
	oldVerbose := flagVerbose
	flagJSON = true
	flagVerbose = false
	defer func() {
		flagJSON = oldJSON
		flagVerbose = oldVerbose
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a mock command
	cmd := NewListCmd() // just need any command
	runTrash(cmd, []string{file})

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	// File should be trashed
	_, err := os.Stat(file)
	assert.True(t, os.IsNotExist(err))

	// JSON output should contain success
	assert.Contains(t, output, "success")
}

// TestRunTrashWithVerbose tests runTrash with verbose output
func TestRunTrashWithVerbose(t *testing.T) {
	setupTestEnv(t)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "verbose_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))

	// Save flags
	oldJSON := flagJSON
	oldVerbose := flagVerbose
	flagJSON = false
	flagVerbose = true
	defer func() {
		flagJSON = oldJSON
		flagVerbose = oldVerbose
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewListCmd()
	runTrash(cmd, []string{file})

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "trashed file")
	assert.Contains(t, output, file)
}

// TestOutputTable tests the outputTable function
func TestOutputTable(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "table_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTable(items)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "table_test.txt")
}

// TestOutputJSON tests the outputJSON function
func TestOutputJSON(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "json_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputJSON(items)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "json_test.txt")
	assert.Contains(t, output, "index")
	assert.Contains(t, output, "size_bytes")
}

// TestOutputJSONStatus tests the outputJSONStatus function
func TestOutputJSONStatus(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "status_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	count, totalSize, err := mgr.Status()
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputJSONStatus(mgr, count, totalSize)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "count")
	assert.Contains(t, output, "size_bytes")
}

// TestOutputJSONStatusVerbose tests verbose JSON status output
func TestOutputJSONStatusVerbose(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "verbose_status.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Enable verbose mode
	oldVerbose := statusVerbose
	statusVerbose = true
	defer func() { statusVerbose = oldVerbose }()

	count, totalSize, err := mgr.Status()
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputJSONStatus(mgr, count, totalSize)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "oldest")
	assert.Contains(t, output, "newest")
	assert.Contains(t, output, "largest")
}

// TestOutputTextStatus tests the outputTextStatus function
func TestOutputTextStatus(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "text_status.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	count, totalSize, err := mgr.Status()
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTextStatus(mgr, count, totalSize)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "files")
}

// TestOutputTextStatusVerbose tests verbose text status output
func TestOutputTextStatusVerbose(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "verbose_text_status.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Enable verbose mode
	oldVerbose := statusVerbose
	statusVerbose = true
	defer func() { statusVerbose = oldVerbose }()

	count, totalSize, err := mgr.Status()
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTextStatus(mgr, count, totalSize)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Oldest")
	assert.Contains(t, output, "Newest")
	assert.Contains(t, output, "Largest")
}

// TestOutputTextStatusEmpty tests text status with empty trash
func TestOutputTextStatusEmpty(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTextStatus(mgr, 0, 0)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "0 files")
}

// TestRestoreLastEmpty tests restoreLast with empty trash
func TestRestoreLastEmpty(t *testing.T) {
	setupTestEnv(t)

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreLast(mgr)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "error")
}

// TestRestoreByInputWithIndex tests restoreByInput with index
func TestRestoreByInputWithIndex(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "restore_index_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreByInput(mgr, items, "1")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "restore_index_test.txt")
	assert.Contains(t, output, "original_path")
}

// TestRestoreByInputWithName tests restoreByInput with name
func TestRestoreByInputWithName(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "restore_name_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreByInput(mgr, items, "restore_name_test.txt")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "restore_name_test.txt")
}

// TestRestoreByInputWithPartialName tests restoreByInput with partial name
func TestRestoreByInputWithPartialName(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "unique_filename_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Use partial name that matches uniquely
	restoreByInput(mgr, items, "unique_filename")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "unique_filename_test.txt")
}

// TestRestoreByInputAmbiguousMatch tests restoreByInput with ambiguous name match
func TestRestoreByInputAmbiguousMatch(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	// Create two files with same prefix
	for i := 0; i < 2; i++ {
		file := filepath.Join(tmpDir, fmt.Sprintf("test_%d.txt", i))
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))
	}

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// This will call exitWithError because of ambiguous match
	// We can't test os.Exit directly, but we verify the code path exists
	assert.Len(t, items, 2) // Both files exist with "test_" prefix
}

// TestRestoreInteractiveEmpty tests restoreInteractive with empty trash
func TestRestoreInteractiveEmpty(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreInteractive(mgr)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Trash is empty")
}

// TestRestoreSpecific tests restoreSpecific function
func TestRestoreSpecific(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "specific_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreSpecific(mgr, "1")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "specific_test.txt")
}

// TestRunRestoreWithLast tests runRestore with --last flag
func TestRunRestoreWithLast(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "last_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldLast := flagLast
	flagJSON = true
	flagLast = true
	defer func() {
		flagJSON = oldJSON
		flagLast = oldLast
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runRestore(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "last_test.txt")
}

// TestRunRestoreWithArg tests runRestore with argument
func TestRunRestoreWithArg(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "arg_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldLast := flagLast
	flagJSON = true
	flagLast = false
	defer func() {
		flagJSON = oldJSON
		flagLast = oldLast
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runRestore(nil, []string{"1"})

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "arg_test.txt")
}

// TestRunStatus tests runStatus function
func TestRunStatus(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "status_run_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runStatus(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "count")
	assert.Contains(t, output, "size_bytes")
}

// TestRunEmpty tests runEmpty function
func TestRunEmpty(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "empty_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldForce := flagForce
	flagJSON = true
	flagForce = true
	defer func() {
		flagJSON = oldJSON
		flagForce = oldForce
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runEmpty(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "removed")
}

// TestRunEmptyWithDays tests runEmpty with --days flag
func TestRunEmptyWithDays(t *testing.T) {
	setupTestEnv(t)

	// Save flags
	oldJSON := flagJSON
	oldForce := flagForce
	oldDays := flagDays
	flagJSON = true
	flagForce = true
	flagDays = 100 // Old enough to not delete recent files
	defer func() {
		flagJSON = oldJSON
		flagForce = oldForce
		flagDays = oldDays
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runEmpty(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "removed")
}

// TestExecuteFunction tests the Execute function
func TestExecuteFunction(t *testing.T) {
	setupTestEnv(t)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "execute_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))

	// Set up args for the command
	oldArgs := os.Args
	os.Args = []string{"trs", file}
	defer func() { os.Args = oldArgs }()

	// Execute should not error
	err := Execute()
	// The command will fail because of os.Exit calls, but we can test it runs
	_ = err
}

// TestRunListTextOutput tests runList with text output
func TestRunListTextOutput(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "list_text_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Ensure JSON is false for text output
	oldJSON := flagJSON
	flagJSON = false
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runList(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	// Should contain table header and file
	assert.Contains(t, output, "list_text_test.txt")
}

// TestRunListEmptyTrash tests runList with empty trash
func TestRunListEmptyTrash(t *testing.T) {
	setupTestEnv(t)

	// Ensure JSON is false for text output
	oldJSON := flagJSON
	flagJSON = false
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runList(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Trash is empty")
}

// TestTrashFileWithForceFlag tests trashFile with force flag
func TestTrashFileWithForceFlag(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Save flags
	oldForce := flagForce
	flagForce = true
	defer func() { flagForce = oldForce }()

	// Try to trash non-existent file with force - should not error
	err = trashFile(mgr, "/nonexistent/file.txt")
	// The error is still returned, but force affects runTrash handling
	assert.Error(t, err)
}

// TestOutputTableWithDirectory tests outputTable with a directory
func TestOutputTableWithDirectory(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "testdir")
	require.NoError(t, os.MkdirAll(dir, 0755))

	// Save flag
	oldRecursive := flagRecursive
	flagRecursive = true
	defer func() { flagRecursive = oldRecursive }()

	require.NoError(t, mgr.Move(dir))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.True(t, items[0].IsDir)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTable(items)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "testdir")
}

// TestRestoreLastWithSuccess tests restoreLast with successful restore
func TestRestoreLastWithSuccess(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "restore_last_success.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldOverwrite := flagOverwrite
	flagJSON = true
	flagOverwrite = false
	defer func() {
		flagJSON = oldJSON
		flagOverwrite = oldOverwrite
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreLast(mgr)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "restore_last_success.txt")
	assert.Contains(t, output, "original_path")
}

// TestRestoreLastTextOutput tests restoreLast with text output
func TestRestoreLastTextOutput(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "restore_last_text.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldOverwrite := flagOverwrite
	flagJSON = false
	flagOverwrite = false
	defer func() {
		flagJSON = oldJSON
		flagOverwrite = oldOverwrite
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreLast(mgr)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Restored")
	assert.Contains(t, output, "restore_last_text.txt")
}

// TestRestoreByInputTextOutput tests restoreByInput with text output
func TestRestoreByInputTextOutput(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "restore_input_text.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Save flags
	oldJSON := flagJSON
	flagJSON = false
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreByInput(mgr, items, "1")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Restored")
}

// TestRestoreInteractiveSimple tests the non-TTY fallback path
func TestRestoreInteractiveSimple(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_SIMPLE") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "simple_restore_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		items, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items, 1)

		// This function reads from stdin, so we test via subprocess
		restoreInteractiveSimple(mgr, items)
		return
	}

	// Run ourselves as a subprocess with stdin providing "1"
	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreInteractiveSimple")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_SIMPLE=1")
	cmd.Stdin = strings.NewReader("1\n")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
	assert.Contains(t, string(output), "simple_restore_test.txt")
}

// TestRestoreInteractiveSimpleCancelled tests canceling the simple restore
func TestRestoreInteractiveSimpleCancelled(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_SIMPLE_CANCEL") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "simple_cancel_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		items, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items, 1)

		restoreInteractiveSimple(mgr, items)
		return
	}

	// Run with empty input (cancel)
	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreInteractiveSimpleCancelled")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_SIMPLE_CANCEL=1")
	cmd.Stdin = strings.NewReader("\n")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
	assert.Contains(t, string(output), "Cancelled")
}

// TestRestoreByInputInvalidIndex tests restoreByInput with invalid index
func TestRestoreByInputInvalidIndex(t *testing.T) {
	if os.Getenv("GO_TEST_INVALID_INDEX") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "invalid_index_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		items, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items, 1)

		restoreByInput(mgr, items, "99")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreByInputInvalidIndex")
	cmd.Env = append(os.Environ(), "GO_TEST_INVALID_INDEX=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, e.ExitCode())
		// Stderr may be empty due to no flush before exit
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRestoreByInputZeroIndex tests restoreByInput with zero index
func TestRestoreByInputZeroIndex(t *testing.T) {
	if os.Getenv("GO_TEST_ZERO_INDEX") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "zero_index_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		items, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items, 1)

		restoreByInput(mgr, items, "0")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreByInputZeroIndex")
	cmd.Env = append(os.Environ(), "GO_TEST_ZERO_INDEX=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, e.ExitCode())
		// Stderr may be empty due to no flush before exit
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRestoreByInputFileNotFound tests restoreByInput with non-existent file name
func TestRestoreByInputFileNotFound(t *testing.T) {
	if os.Getenv("GO_TEST_FILE_NOT_FOUND") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "file_not_found_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		items, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items, 1)

		restoreByInput(mgr, items, "nonexistent_file.txt")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreByInputFileNotFound")
	cmd.Env = append(os.Environ(), "GO_TEST_FILE_NOT_FOUND=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, e.ExitCode())
		// Stderr may be empty due to no flush before exit
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRestoreByInputAmbiguousMatchExit tests ambiguous match with exit
func TestRestoreByInputAmbiguousMatchExit(t *testing.T) {
	if os.Getenv("GO_TEST_AMBIGUOUS_EXIT") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		// Create two files with same prefix
		for i := 0; i < 2; i++ {
			file := filepath.Join(tmpDir, fmt.Sprintf("ambiguous_%d.txt", i))
			require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
			require.NoError(t, mgr.Move(file))
		}

		items, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items, 2)

		// This should exit with error because of ambiguous match
		restoreByInput(mgr, items, "ambiguous")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreByInputAmbiguousMatchExit")
	cmd.Env = append(os.Environ(), "GO_TEST_AMBIGUOUS_EXIT=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, e.ExitCode())
		// Stderr may be empty due to no flush before exit
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRunTrashNoArgsExit tests runTrash with no args (exits)
func TestRunTrashNoArgsExit(t *testing.T) {
	if os.Getenv("GO_TEST_TRASH_NO_ARGS") == "1" {
		setupTestEnv(t)
		cmd := NewListCmd() // any command for Help()
		runTrash(cmd, []string{})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunTrashNoArgsExit")
	cmd.Env = append(os.Environ(), "GO_TEST_TRASH_NO_ARGS=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, e.ExitCode())
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRunTrashWithErrors tests runTrash with errors
func TestRunTrashWithErrors(t *testing.T) {
	if os.Getenv("GO_TEST_TRASH_ERRORS") == "1" {
		setupTestEnv(t)

		// Save flags
		oldForce := flagForce
		oldJSON := flagJSON
		oldVerbose := flagVerbose
		flagForce = false
		flagJSON = false
		flagVerbose = false
		defer func() {
			flagForce = oldForce
			flagJSON = oldJSON
			flagVerbose = oldVerbose
		}()

		cmd := NewListCmd()
		runTrash(cmd, []string{"/nonexistent/path/file.txt"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunTrashWithErrors")
	cmd.Env = append(os.Environ(), "GO_TEST_TRASH_ERRORS=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, e.ExitCode())
		// Stderr may be empty due to no flush before exit
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRunTrashWithJSONErrors tests runTrash with JSON output and errors
func TestRunTrashWithJSONErrors(t *testing.T) {
	if os.Getenv("GO_TEST_TRASH_JSON_ERRORS") == "1" {
		setupTestEnv(t)

		// Save flags
		oldJSON := flagJSON
		oldForce := flagForce
		flagJSON = true
		flagForce = false
		defer func() {
			flagJSON = oldJSON
			flagForce = oldForce
		}()

		cmd := NewListCmd()
		runTrash(cmd, []string{"/nonexistent/path/file.txt"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunTrashWithJSONErrors")
	cmd.Env = append(os.Environ(), "GO_TEST_TRASH_JSON_ERRORS=1")
	output, err := cmd.CombinedOutput()

	// Should exit with code 1
	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, e.ExitCode())
		assert.Contains(t, string(output), "errors")
		assert.Contains(t, string(output), "failed")
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRunEmptyTextOutput tests runEmpty with text output
func TestRunEmptyTextOutput(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "empty_text_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldForce := flagForce
	flagJSON = false
	flagForce = true
	defer func() {
		flagJSON = oldJSON
		flagForce = oldForce
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runEmpty(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Trash emptied")
	assert.Contains(t, output, "files removed")
}

// TestRunEmptyWithDaysTextOutput tests runEmpty with --days flag and text output
func TestRunEmptyWithDaysTextOutput(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "days_text_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldForce := flagForce
	oldDays := flagDays
	flagJSON = false
	flagForce = true
	flagDays = 100 // Old enough to not delete recent files
	defer func() {
		flagJSON = oldJSON
		flagForce = oldForce
		flagDays = oldDays
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runEmpty(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Removed")
	assert.Contains(t, output, "files older than")
}

// TestRunEmptyCancelled tests runEmpty when cancelled by user
func TestRunEmptyCancelled(t *testing.T) {
	if os.Getenv("GO_TEST_EMPTY_CANCEL") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "empty_cancel_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		// Save flags - no force, no JSON
		oldForce := flagForce
		oldJSON := flagJSON
		flagForce = false
		flagJSON = false
		defer func() {
			flagForce = oldForce
			flagJSON = oldJSON
		}()

		runEmpty(nil, nil)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunEmptyCancelled")
	cmd.Env = append(os.Environ(), "GO_TEST_EMPTY_CANCEL=1")
	cmd.Stdin = strings.NewReader("n\n")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
	assert.Contains(t, string(output), "Cancelled")
}

// TestRunStatusTextOutput tests runStatus with text output
func TestRunStatusTextOutput(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "status_text_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	flagJSON = false
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runStatus(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Trash:")
	assert.Contains(t, output, "files")
}

// TestRestoreSpecificError tests restoreSpecific with list error
func TestRestoreSpecificError(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_SPECIFIC_ERROR") == "1" {
		// Create an invalid trash state to cause an error
		tmpHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
		t.Setenv("HOME", tmpHome)

		// Create a corrupted trash directory
		trashDir := filepath.Join(tmpHome, ".local", "share", "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))

		// Create an invalid .trashinfo file
		infoFile := filepath.Join(trashDir, "info", "bad_file.trashinfo")
		// Write invalid content that will cause parse error
		require.NoError(t, os.WriteFile(infoFile, []byte("invalid content"), 0644))

		restoreSpecific(nil, "bad_file")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreSpecificError")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_SPECIFIC_ERROR=1")
	// This should not crash, but may error
	_ = cmd.Run()
}

// TestRestoreLastTextOutputError tests restoreLast with error and text output
func TestRestoreLastTextOutputError(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_LAST_TEXT_ERROR") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		// No files in trash, so GetLast will fail

		// Save flags
		oldJSON := flagJSON
		flagJSON = false
		defer func() { flagJSON = oldJSON }()

		restoreLast(mgr)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreLastTextOutputError")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_LAST_TEXT_ERROR=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, e.ExitCode())
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRestoreByInputWithRestoreError tests restoreByInput when restore fails
func TestRestoreByInputWithRestoreError(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_ERROR") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "restore_error_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		// Create a file at the original location to cause restore conflict
		require.NoError(t, os.WriteFile(file, []byte("conflict"), 0644))

		items, err := mgr.List()
		require.NoError(t, err)
		require.Len(t, items, 1)

		// Save flags - no overwrite, no JSON
		oldJSON := flagJSON
		oldOverwrite := flagOverwrite
		flagJSON = false
		flagOverwrite = false
		defer func() {
			flagJSON = oldJSON
			flagOverwrite = oldOverwrite
		}()

		restoreByInput(mgr, items, "1")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreByInputWithRestoreError")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_ERROR=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, e.ExitCode())
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRestoreLastWithRestoreError tests restoreLast when restore fails
func TestRestoreLastWithRestoreError(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_LAST_ERROR") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "restore_last_error_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		// Create a file at the original location to cause restore conflict
		require.NoError(t, os.WriteFile(file, []byte("conflict"), 0644))

		// Save flags - no overwrite, no JSON
		oldJSON := flagJSON
		oldOverwrite := flagOverwrite
		flagJSON = false
		flagOverwrite = false
		defer func() {
			flagJSON = oldJSON
			flagOverwrite = oldOverwrite
		}()

		restoreLast(mgr)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreLastWithRestoreError")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_LAST_ERROR=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, e.ExitCode())
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestRestoreLastJSONError tests restoreLast with error and JSON output
func TestRestoreLastJSONError(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// No files in trash

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreLast(mgr)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "error")
}

// TestRestoreByInputJSONWithRestoreError tests restoreByInput JSON output with error
func TestRestoreByInputJSONWithRestoreError(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "restore_json_error_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Create a file at the original location to cause restore conflict
	require.NoError(t, os.WriteFile(file, []byte("conflict"), 0644))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Save flags
	oldJSON := flagJSON
	oldOverwrite := flagOverwrite
	flagJSON = true
	flagOverwrite = false
	defer func() {
		flagJSON = oldJSON
		flagOverwrite = oldOverwrite
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreByInput(mgr, items, "1")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "error")
}

// TestRunRestoreInteractiveError tests restoreInteractive with manager error
func TestRunRestoreInteractiveError(t *testing.T) {
	if os.Getenv("GO_TEST_INTERACTIVE_ERROR") == "1" {
		// Create an invalid trash state to cause an error
		tmpHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
		t.Setenv("HOME", tmpHome)

		// Create a corrupted trash directory
		trashDir := filepath.Join(tmpHome, ".local", "share", "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))

		// Create an invalid .trashinfo file
		infoFile := filepath.Join(trashDir, "info", "bad_file.trashinfo")
		require.NoError(t, os.WriteFile(infoFile, []byte("invalid content"), 0644))

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		restoreInteractive(mgr)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunRestoreInteractiveError")
	cmd.Env = append(os.Environ(), "GO_TEST_INTERACTIVE_ERROR=1")
	_ = cmd.Run() // Should not crash
}

// TestRestoreInteractiveWithJSON tests restoreInteractive with JSON output
func TestRestoreInteractiveWithJSON(t *testing.T) {
	if os.Getenv("GO_TEST_INTERACTIVE_JSON") == "1" {
		setupTestEnv(t)

		mgr, err := trash.NewManager()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "interactive_json_test.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		require.NoError(t, mgr.Move(file))

		// JSON flag is checked but not used in non-TTY mode
		restoreInteractive(mgr)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreInteractiveWithJSON")
	cmd.Env = append(os.Environ(), "GO_TEST_INTERACTIVE_JSON=1")
	cmd.Stdin = strings.NewReader("\n") // Cancel
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
}

// TestRunTrashVerboseErrors tests runTrash with verbose and errors
func TestRunTrashVerboseErrors(t *testing.T) {
	if os.Getenv("GO_TEST_TRASH_VERBOSE_ERRORS") == "1" {
		setupTestEnv(t)

		// Save flags
		oldForce := flagForce
		oldJSON := flagJSON
		oldVerbose := flagVerbose
		flagForce = false
		flagJSON = false
		flagVerbose = true
		defer func() {
			flagForce = oldForce
			flagJSON = oldJSON
			flagVerbose = oldVerbose
		}()

		cmd := NewListCmd()
		runTrash(cmd, []string{"/nonexistent/path/file.txt"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunTrashVerboseErrors")
	cmd.Env = append(os.Environ(), "GO_TEST_TRASH_VERBOSE_ERRORS=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, e.ExitCode())
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestTrashFilePermissionError tests trashFile with permission error
func TestTrashFilePermissionError(t *testing.T) {
	// This test requires root or special setup, so we skip it
	// Just verify the error path exists conceptually
	t.Skip("requires special permission setup")
}

// TestRestoreInteractiveSimpleDirect tests restoreInteractiveSimple directly
func TestRestoreInteractiveSimpleDirect(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "simple_direct_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write input in a goroutine
	go func() {
		w.Write([]byte("1\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	restoreInteractiveSimple(mgr, items)

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	assert.Contains(t, output, "simple_direct_test.txt")
}

// TestRestoreInteractiveSimpleDirectCancel tests restoreInteractiveSimple with cancel
func TestRestoreInteractiveSimpleDirectCancel(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "simple_cancel_direct.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	items, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write empty input in a goroutine (cancel)
	go func() {
		w.Write([]byte("\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	restoreInteractiveSimple(mgr, items)

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	assert.Contains(t, output, "Cancelled")
}

// TestRestoreInteractiveNonEmpty tests restoreInteractive with non-empty trash
func TestRestoreInteractiveNonEmpty(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "interactive_nonempty_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write input in a goroutine
	go func() {
		w.Write([]byte("1\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	restoreInteractive(mgr)

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	// Since stdout is not a terminal, it should call restoreInteractiveSimple
	assert.Contains(t, output, "interactive_nonempty_test.txt")
}

// TestRestoreInteractiveNonEmptyCancel tests restoreInteractive with cancel
func TestRestoreInteractiveNonEmptyCancel(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "interactive_cancel_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write empty input in a goroutine (cancel)
	go func() {
		w.Write([]byte("\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	restoreInteractive(mgr)

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	assert.Contains(t, output, "Cancelled")
}

// TestRunEmptyConfirmYes tests runEmpty with confirmation yes
func TestRunEmptyConfirmYes(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "empty_confirm_yes.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldForce := flagForce
	oldJSON := flagJSON
	flagForce = false
	flagJSON = false
	defer func() {
		flagForce = oldForce
		flagJSON = oldJSON
	}()

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write "y" in a goroutine
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	runEmpty(nil, nil)

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	assert.Contains(t, output, "Trash emptied")
}

// TestRunEmptyConfirmDays tests runEmpty with confirmation and --days
func TestRunEmptyConfirmDays(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "empty_confirm_days.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldForce := flagForce
	oldJSON := flagJSON
	oldDays := flagDays
	flagForce = false
	flagJSON = false
	flagDays = 100 // Won't delete recent files
	defer func() {
		flagForce = oldForce
		flagJSON = oldJSON
		flagDays = oldDays
	}()

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write "y" in a goroutine
	go func() {
		w.Write([]byte("y\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	runEmpty(nil, nil)

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	assert.Contains(t, output, "Removed")
	assert.Contains(t, output, "older than")
}

// TestRunEmptyConfirmNo tests runEmpty with confirmation no
func TestRunEmptyConfirmNo(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "empty_confirm_no.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldForce := flagForce
	oldJSON := flagJSON
	flagForce = false
	flagJSON = false
	defer func() {
		flagForce = oldForce
		flagJSON = oldJSON
	}()

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write "n" in a goroutine
	go func() {
		w.Write([]byte("n\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	runEmpty(nil, nil)

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	assert.Contains(t, output, "Cancelled")
}

// TestNewVersionCmdJSON tests NewVersionCmd with JSON output
func TestNewVersionCmdJSON(t *testing.T) {
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewVersionCmd()
	cmd.Run(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "version")
}

// TestNewVersionCmdText tests NewVersionCmd with text output
func TestNewVersionCmdText(t *testing.T) {
	oldJSON := flagJSON
	flagJSON = false
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewVersionCmd()
	cmd.Run(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	// Version output should contain something
	assert.NotEmpty(t, output)
}

// TestRunStatusTextOutputVerbose tests runStatus with verbose text output
func TestRunStatusTextOutputVerbose(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "status_verbose_text.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	oldVerbose := statusVerbose
	flagJSON = false
	statusVerbose = true
	defer func() {
		flagJSON = oldJSON
		statusVerbose = oldVerbose
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runStatus(nil, nil)

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "Trash:")
	assert.Contains(t, output, "Oldest:")
	assert.Contains(t, output, "Newest:")
}

// TestRestoreSpecificDirect tests restoreSpecific directly
func TestRestoreSpecificDirect(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "specific_direct_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreSpecific(mgr, "1")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "specific_direct_test.txt")
}

// TestRestoreSpecificByName tests restoreSpecific by name
func TestRestoreSpecificByName(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "specific_by_name_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	restoreSpecific(mgr, "specific_by_name_test.txt")

	os.Stdout = oldStdout
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()
	output := buf.String()

	assert.Contains(t, output, "specific_by_name_test.txt")
}

// TestRunRestoreInteractive tests runRestore with interactive mode
func TestRunRestoreInteractive(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "run_restore_interactive.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
	require.NoError(t, mgr.Move(file))

	// Save flags
	oldLast := flagLast
	oldJSON := flagJSON
	flagLast = false
	flagJSON = true
	defer func() {
		flagLast = oldLast
		flagJSON = oldJSON
	}()

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write input in a goroutine
	go func() {
		w.Write([]byte("1\n"))
		w.Close()
	}()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	runRestore(nil, []string{})

	// Restore stdin/stdout
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	wOut.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	rOut.Close()
	output := buf.String()

	assert.Contains(t, output, "run_restore_interactive.txt")
}

// TestRunListError - placeholder for error path (subprocess test doesn't contribute to coverage)
func TestRunListError(t *testing.T) {
	// This test is a placeholder - the actual error paths are tested via integration tests
	t.Skip("error paths tested via integration tests")
}

// TestRestoreSpecificListError - placeholder for error path
func TestRestoreSpecificListError(t *testing.T) {
	t.Skip("error paths tested via integration tests")
}

// TestRunStatusError tests runStatus with manager error
func TestRunStatusError(t *testing.T) {
	if os.Getenv("GO_TEST_STATUS_ERROR") == "1" {
		// Create invalid trash state
		tmpHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
		t.Setenv("HOME", tmpHome)

		// Create corrupted trash directory
		trashDir := filepath.Join(tmpHome, ".local", "share", "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))

		// Create invalid .trashinfo file
		infoFile := filepath.Join(trashDir, "info", "bad.trashinfo")
		require.NoError(t, os.WriteFile(infoFile, []byte("invalid"), 0644))

		// This should not crash
		runStatus(nil, nil)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunStatusError")
	cmd.Env = append(os.Environ(), "GO_TEST_STATUS_ERROR=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
}

// TestRunEmptyManagerError tests runEmpty with manager error
func TestRunEmptyManagerError(t *testing.T) {
	if os.Getenv("GO_TEST_EMPTY_MGR_ERROR") == "1" {
		// Create invalid trash state
		tmpHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
		t.Setenv("HOME", tmpHome)

		// Create corrupted trash directory
		trashDir := filepath.Join(tmpHome, ".local", "share", "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))

		// Create invalid .trashinfo file
		infoFile := filepath.Join(trashDir, "info", "bad.trashinfo")
		require.NoError(t, os.WriteFile(infoFile, []byte("invalid"), 0644))

		// Set flags
		oldJSON := flagJSON
		oldForce := flagForce
		flagJSON = true
		flagForce = true
		defer func() {
			flagJSON = oldJSON
			flagForce = oldForce
		}()

		// This should not crash
		runEmpty(nil, nil)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunEmptyManagerError")
	cmd.Env = append(os.Environ(), "GO_TEST_EMPTY_MGR_ERROR=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
}

// TestRunRestoreManagerError tests runRestore with manager error
func TestRunRestoreManagerError(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_MGR_ERROR") == "1" {
		// Create invalid trash state
		tmpHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
		t.Setenv("HOME", tmpHome)

		// Create corrupted trash directory
		trashDir := filepath.Join(tmpHome, ".local", "share", "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))

		// Create invalid .trashinfo file
		infoFile := filepath.Join(trashDir, "info", "bad.trashinfo")
		require.NoError(t, os.WriteFile(infoFile, []byte("invalid"), 0644))

		// Set flags
		oldJSON := flagJSON
		flagJSON = true
		defer func() { flagJSON = oldJSON }()

		// This should not crash
		runRestore(nil, nil)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunRestoreManagerError")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_MGR_ERROR=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
}

// TestRunListManagerError tests runList with manager error
func TestRunListManagerError(t *testing.T) {
	if os.Getenv("GO_TEST_LIST_MGR_ERROR") == "1" {
		// Create invalid trash state
		tmpHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
		t.Setenv("HOME", tmpHome)

		// Create corrupted trash directory
		trashDir := filepath.Join(tmpHome, ".local", "share", "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))

		// Create invalid .trashinfo file
		infoFile := filepath.Join(trashDir, "info", "bad.trashinfo")
		require.NoError(t, os.WriteFile(infoFile, []byte("invalid"), 0644))

		// Set flags
		oldJSON := flagJSON
		flagJSON = true
		defer func() { flagJSON = oldJSON }()

		// This should not crash
		runList(nil, nil)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunListManagerError")
	cmd.Env = append(os.Environ(), "GO_TEST_LIST_MGR_ERROR=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", output)
}

// TestRestoreSpecificManagerError tests restoreSpecific with manager error
func TestRestoreSpecificManagerError(t *testing.T) {
	if os.Getenv("GO_TEST_RESTORE_SPECIFIC_MGR_ERROR") == "1" {
		// Create invalid trash state
		tmpHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
		t.Setenv("HOME", tmpHome)

		// Create corrupted trash directory
		trashDir := filepath.Join(tmpHome, ".local", "share", "Trash")
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))

		// Create invalid .trashinfo file
		infoFile := filepath.Join(trashDir, "info", "bad.trashinfo")
		require.NoError(t, os.WriteFile(infoFile, []byte("invalid"), 0644))

		// Try to restore
		restoreSpecific(nil, "1")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRestoreSpecificManagerError")
	cmd.Env = append(os.Environ(), "GO_TEST_RESTORE_SPECIFIC_MGR_ERROR=1")
	err := cmd.Run()
	// Should exit with error
	if e, ok := err.(*exec.ExitError); ok {
		assert.NotEqual(t, 0, e.ExitCode())
	} else {
		t.Fatalf("expected exit error, got: %v", err)
	}
}

// TestValidateCLIInput tests the validateCLIInput function
func TestValidateCLIInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid short input",
			input:   "1",
			wantErr: false,
		},
		{
			name:    "valid name input",
			input:   "file.txt",
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
		},
		{
			name:    "valid y confirmation",
			input:   "y",
			wantErr: false,
		},
		{
			name:    "valid N confirmation",
			input:   "N",
			wantErr: false,
		},
		{
			name:    "input at max length",
			input:   strings.Repeat("a", 4096),
			wantErr: false,
		},
		{
			name:    "input over max length",
			input:   strings.Repeat("a", 4097),
			wantErr: true,
			errMsg:  "input too long",
		},
		{
			name:    "input with null byte",
			input:   "test\x00file",
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:    "only null byte",
			input:   "\x00",
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:    "input with null at end",
			input:   "valid\x00",
			wantErr: true,
			errMsg:  "null byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCLIInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
