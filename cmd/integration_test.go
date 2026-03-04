package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amakeenk/trs/internal/trash"
	"github.com/stretchr/testify/require"
)

// Helper to capture stdout
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	os.Stdout = oldStdout
	w.Close()

	var buf []byte
	buf = make([]byte, 1024)
	n, _ := r.Read(buf)
	r.Close()
	return string(buf[:n])
}

// Test for empty trash list
func TestRunListEmpty(t *testing.T) {
	setupTestEnv(t)

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	output := captureStdout(t, func() {
		runList(nil, nil)
	})

	require.Contains(t, output, "[]")
}

// Test JSON output for restore --last
func TestRestoreLastJSON(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Add a file
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))
	require.NoError(t, mgr.Move(file))

	// Test with JSON flag
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	oldLast := flagLast
	flagLast = true
	defer func() { flagLast = oldLast }()

	output := captureStdout(t, func() {
		restoreLast(mgr)
	})

	require.Contains(t, output, "test.txt")
	require.Contains(t, output, "original_path")
}

// Test JSON output for empty command
func TestEmptyJSON(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Add a file
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))
	require.NoError(t, mgr.Move(file))

	// Test with JSON flag and force flag to skip confirmation
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	oldForce := flagForce
	flagForce = true
	defer func() { flagForce = oldForce }()

	output := captureStdout(t, func() {
		runEmpty(nil, nil)
	})

	require.Contains(t, output, "removed")
}

// Test empty trash is empty message
func TestEmptyAlreadyEmpty(t *testing.T) {
	setupTestEnv(t)

	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	output := captureStdout(t, func() {
		runEmpty(nil, nil)
	})

	require.Contains(t, output, "Trash is empty")
}

// Test status JSON output
func TestStatusJSON(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Add a file
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello world"), 0644))
	require.NoError(t, mgr.Move(file))

	// Test with JSON flag
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	output := captureStdout(t, func() {
		runStatus(nil, nil)
	})

	require.Contains(t, output, "count")
	require.Contains(t, output, "size_bytes")
}

// Test status verbose JSON output
func TestStatusVerboseJSON(t *testing.T) {
	setupTestEnv(t)

	mgr, err := trash.NewManager()
	require.NoError(t, err)

	// Add a file
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello world"), 0644))
	require.NoError(t, mgr.Move(file))

	// Test with JSON and verbose flags
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	oldVerbose := statusVerbose
	statusVerbose = true
	defer func() { statusVerbose = oldVerbose }()

	output := captureStdout(t, func() {
		runStatus(nil, nil)
	})

	require.Contains(t, output, "oldest")
	require.Contains(t, output, "newest")
	require.Contains(t, output, "largest")
}

// Test version JSON output
func TestVersionJSON(t *testing.T) {
	oldJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = oldJSON }()

	output := captureStdout(t, func() {
		cmd := NewVersionCmd()
		cmd.Run(nil, nil)
	})

	require.Contains(t, output, "version")
}
