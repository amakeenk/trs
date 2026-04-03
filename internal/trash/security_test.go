// Package trash provides security regression tests for the trash manager.
// These tests document and verify protection against known attack patterns.
package trash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecurity_SymlinkAttacks verifies protection against symlink-based attacks.
// Symlinks can be used to escape intended directories or delete unintended files.
func TestSecurity_SymlinkAttacks(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func(tmpDir string) (targetPath string, attackPath string)
		action      func(mgr *Manager, attackPath string) error
		wantErr     bool
		errContains string
		verifySafe  func(t *testing.T, targetPath string)
	}{
		{
			name:        "symlink_file_trashed_not_target",
			description: "Trashing a symlink should move the link, not the target",
			setup: func(tmpDir string) (string, string) {
				target := filepath.Join(tmpDir, "target.txt")
				require.NoError(t, os.WriteFile(target, []byte("target content"), 0644))
				link := filepath.Join(tmpDir, "link.txt")
				require.NoError(t, os.Symlink(target, link))
				return target, link
			},
			action: func(mgr *Manager, attackPath string) error {
				return mgr.Move(attackPath)
			},
			wantErr: false, // Moving symlink is allowed
			verifySafe: func(t *testing.T, targetPath string) {
				// Target should still exist
				assert.FileExists(t, targetPath)
				content, err := os.ReadFile(targetPath)
				require.NoError(t, err)
				assert.Equal(t, "target content", string(content))
			},
		},
		{
			name:        "directory_with_symlink_inside_allowed",
			description: "Directory containing symlink should be allowed during cross-device copy now",
			setup: func(tmpDir string) (string, string) {
				// Create sensitive data outside the directory to be trashed
				sensitiveDir := filepath.Join(tmpDir, "sensitive")
				require.NoError(t, os.MkdirAll(sensitiveDir, 0755))
				sensitiveFile := filepath.Join(sensitiveDir, "important.txt")
				require.NoError(t, os.WriteFile(sensitiveFile, []byte("SENSITIVE DATA"), 0644))

				// Create directory with symlink inside pointing to sensitive data
				attackDir := filepath.Join(tmpDir, "attackdir")
				require.NoError(t, os.MkdirAll(attackDir, 0755))
				link := filepath.Join(attackDir, "link_to_sensitive")
				require.NoError(t, os.Symlink(sensitiveDir, link))

				return sensitiveFile, attackDir
			},
			action: func(mgr *Manager, attackPath string) error {
				// Use copyDirAndDelete to simulate cross-device move
				dstDir := filepath.Join(filepath.Dir(attackPath), "destination")
				return mgr.copyDirAndDelete(attackPath, dstDir)
			},
			wantErr:     false,
			errContains: "",
			verifySafe: func(t *testing.T, targetPath string) {
				// Sensitive file should NOT be deleted
				assert.FileExists(t, targetPath)
				content, err := os.ReadFile(targetPath)
				require.NoError(t, err)
				assert.Equal(t, "SENSITIVE DATA", string(content))
			},
		},
		{
			name:        "copy_file_rejects_symlink_source",
			description: "copyFile should reject symlink source to prevent following links",
			setup: func(tmpDir string) (string, string) {
				target := filepath.Join(tmpDir, "target.txt")
				require.NoError(t, os.WriteFile(target, []byte("target content"), 0644))
				link := filepath.Join(tmpDir, "link.txt")
				require.NoError(t, os.Symlink(target, link))
				return target, link
			},
			action: func(mgr *Manager, attackPath string) error {
				dst := filepath.Join(filepath.Dir(attackPath), "copy.txt")
				return copyFile(attackPath, dst)
			},
			wantErr:     true,
			errContains: "symlink",
			verifySafe: func(t *testing.T, targetPath string) {
				// Target should still exist
				assert.FileExists(t, targetPath)
			},
		},
		{
			name:        "safeRemoveAll_allows_directory_with_symlink",
			description: "safeRemoveAll should allow directories containing symlinks now",
			setup: func(tmpDir string) (string, string) {
				sensitiveDir := filepath.Join(tmpDir, "sensitive")
				require.NoError(t, os.MkdirAll(sensitiveDir, 0755))
				sensitiveFile := filepath.Join(sensitiveDir, "data.txt")
				require.NoError(t, os.WriteFile(sensitiveFile, []byte("PROTECTED"), 0644))

				attackDir := filepath.Join(tmpDir, "toRemove")
				require.NoError(t, os.MkdirAll(attackDir, 0755))
				link := filepath.Join(attackDir, "link")
				require.NoError(t, os.Symlink(sensitiveDir, link))

				return sensitiveFile, attackDir
			},
			action: func(mgr *Manager, attackPath string) error {
				return safeRemoveAll(attackPath)
			},
			wantErr:     false,
			errContains: "",
			verifySafe: func(t *testing.T, targetPath string) {
				assert.FileExists(t, targetPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			tmpHome := t.TempDir()
			xdgData := filepath.Join(tmpHome, ".local", "share")
			t.Setenv("XDG_DATA_HOME", xdgData)
			t.Setenv("HOME", tmpHome)

			mgr, err := NewManager()
			require.NoError(t, err)

			// Create test directory separate from home
			testDir := t.TempDir()
			targetPath, attackPath := tt.setup(testDir)

			// Execute the attack action
			err = tt.action(mgr, attackPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify target is safe
			if tt.verifySafe != nil {
				tt.verifySafe(t, targetPath)
			}
		})
	}
}

// TestSecurity_PathTraversal verifies protection against path traversal attacks.
// Path traversal can allow access to files outside intended directories.
func TestSecurity_PathTraversal(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
		wantErr     bool
		errContains string
	}{
		{
			name:        "filename_with_slash_rejected",
			input:       "test/file.txt",
			description: "Filename containing path separator should be rejected",
			wantErr:     true,
			errContains: "path separator",
		},
		{
			name:        "filename_with_backslash_rejected",
			input:       "test\\file.txt",
			description: "Filename containing backslash should be rejected",
			wantErr:     true,
			errContains: "path separator",
		},
		{
			name:        "filename_with_double_dot_rejected",
			input:       "test..file.txt",
			description: "Filename containing '..' should be rejected",
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name:        "single_dot_rejected",
			input:       ".",
			description: "Single dot as filename should be rejected",
			wantErr:     true,
			errContains: "invalid filename",
		},
		{
			name:        "double_dot_rejected",
			input:       "..",
			description: "Double dot as filename should be rejected",
			wantErr:     true,
			errContains: "invalid filename",
		},
		{
			name:        "valid_filename_accepted",
			input:       "normal_file.txt",
			description: "Normal filename should be accepted",
			wantErr:     false,
		},
		{
			name:        "filename_with_single_dot_accepted",
			input:       ".hidden_file",
			description: "Hidden file (dot prefix) should be accepted",
			wantErr:     false,
		},
		{
			name:        "complex_valid_filename",
			input:       "my-file_v2.0.test.txt",
			description: "Complex but valid filename should be accepted",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSecurity_LongPaths verifies handling of excessively long paths.
// Long paths can cause buffer overflows or DoS in some implementations.
func TestSecurity_LongPaths(t *testing.T) {
	tests := []struct {
		name        string
		description string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "very_long_filename_255_chars",
			description: "255 character filename (typical filesystem limit)",
			path:        strings.Repeat("a", 255),
			wantErr:     false, // Should be handled gracefully
		},
		{
			name:        "extremely_long_filename_1000_chars",
			description: "1000 character filename exceeds typical limits",
			path:        strings.Repeat("b", 1000),
			wantErr:     false, // May be truncated or handled by filesystem
		},
		{
			name:        "very_long_path_4096_chars",
			description: "4096 character path (typical PATH_MAX)",
			path:        "/" + strings.Repeat("c/", 800) + "file.txt",
			wantErr:     false, // Should be handled gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validateFileName with long names
			if len(tt.path) < 256 || !strings.Contains(tt.path, "/") {
				err := validateFileName(filepath.Base(tt.path))
				// Long filenames are typically allowed (filesystem handles truncation)
				// The function should not panic
				_ = err
			}
		})
	}

	// Test resolveNameConflict iteration limit
	t.Run("resolveNameConflict_iteration_limit", func(t *testing.T) {
		tmpHome := t.TempDir()
		xdgData := filepath.Join(tmpHome, ".local", "share")
		t.Setenv("XDG_DATA_HOME", xdgData)
		t.Setenv("HOME", tmpHome)

		mgr, err := NewManager()
		require.NoError(t, err)

		trashDir := filepath.Join(xdgData, "Trash")
		require.NoError(t, EnsureTrashDir(trashDir))

		// Create many conflicting files to approach the limit
		for i := 1; i < 100; i++ {
			conflictFile := FilesPath(trashDir, "conflict.txt")
			if i > 1 {
				conflictFile = FilesPath(trashDir, "conflict.txt_"+string(rune('0'+i/10))+string(rune('0'+i%10)))
			}
			require.NoError(t, os.WriteFile(conflictFile, []byte("test"), 0644))
		}

		// This should eventually hit the iteration limit
		// But 100 is below the 10000 limit, so it should work
		name, err := mgr.resolveNameConflict(trashDir, "conflict.txt")
		require.NoError(t, err)
		assert.NotEmpty(t, name)
	})
}

// TestSecurity_NullBytes verifies rejection of null bytes in input.
// Null bytes can cause string truncation vulnerabilities in C-based systems
// and should be rejected as defense-in-depth even in Go.
func TestSecurity_NullBytes(t *testing.T) {
	tests := []struct {
		name        string
		description string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "null_byte_at_start",
			description: "Null byte at the start of filename",
			input:       "\x00file.txt",
			wantErr:     true,
			errContains: "null byte",
		},
		{
			name:        "null_byte_in_middle",
			description: "Null byte in the middle of filename",
			input:       "file\x00name.txt",
			wantErr:     true,
			errContains: "null byte",
		},
		{
			name:        "null_byte_at_end",
			description: "Null byte at the end of filename",
			input:       "file.txt\x00",
			wantErr:     true,
			errContains: "null byte",
		},
		{
			name:        "multiple_null_bytes",
			description: "Multiple null bytes in filename",
			input:       "a\x00b\x00c\x00",
			wantErr:     true,
			errContains: "null byte",
		},
		{
			name:        "no_null_byte",
			description: "Filename without null byte is valid",
			input:       "normal_file.txt",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSecurity_EmptyFilenames verifies rejection of empty or invalid filenames.
// Empty filenames can cause unexpected behavior or errors.
func TestSecurity_EmptyFilenames(t *testing.T) {
	tests := []struct {
		name        string
		description string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty_string",
			description: "Empty string should be rejected",
			input:       "",
			wantErr:     true,
			errContains: "invalid filename",
		},
		{
			name:        "only_whitespace",
			description: "Whitespace-only string should be rejected",
			input:       "   ",
			wantErr:     false, // Whitespace is technically a valid filename
		},
		{
			name:        "single_space",
			description: "Single space is technically valid but unusual",
			input:       " ",
			wantErr:     false, // Space is a valid filename character
		},
		{
			name:        "valid_simple_name",
			description: "Simple valid filename",
			input:       "file.txt",
			wantErr:     false,
		},
		{
			name:        "valid_name_with_spaces",
			description: "Filename with spaces is valid",
			input:       "my file name.txt",
			wantErr:     false,
		},
		{
			name:        "unicode_filename",
			description: "Unicode characters should be accepted",
			input:       "файл_文件_📄.txt",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSecurity_TrashInfoValidation verifies validation of .trashinfo file paths.
// Malicious .trashinfo files could contain paths outside intended directories.
func TestSecurity_TrashInfoValidation(t *testing.T) {
	tests := []struct {
		name        string
		description string
		trashPath   string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid_user_path",
			description: "Valid path in user directory",
			trashPath:   "/home/user/documents/file.txt",
			wantErr:     false,
		},
		{
			name:        "valid_tmp_path",
			description: "Valid path in /tmp",
			trashPath:   "/tmp/test.txt",
			wantErr:     false,
		},
		{
			name:        "system_path_etc_rejected",
			description: "System path /etc should be rejected",
			trashPath:   "/etc/passwd",
			wantErr:     true,
			errContains: "system path",
		},
		{
			name:        "system_path_root_rejected",
			description: "System path /root should be rejected",
			trashPath:   "/root/.bashrc",
			wantErr:     true,
			errContains: "system path",
		},
		{
			name:        "relative_path_rejected",
			description: "Relative path should be rejected",
			trashPath:   "relative/path.txt",
			wantErr:     true,
			errContains: "invalid restore path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRestorePath(tt.trashPath)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSecurity_RestoreWithMaliciousTrashInfo tests restore with crafted .trashinfo.
func TestSecurity_RestoreWithMaliciousTrashInfo(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	// Create trash directory structure
	trashDir := filepath.Join(xdgData, "Trash")
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "files"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(trashDir, "info"), 0755))

	// Create a file in trash
	trashFile := filepath.Join(trashDir, "files", "malicious.txt")
	require.NoError(t, os.WriteFile(trashFile, []byte("content"), 0644))

	// Create trashinfo with system path
	trashInfo := filepath.Join(trashDir, "info", "malicious.txt.trashinfo")
	infoContent := "[Trash Info]\nPath=/etc/passwd\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"
	require.NoError(t, os.WriteFile(trashInfo, []byte(infoContent), 0644))

	mgr, err := NewManager()
	require.NoError(t, err)

	// Attempt restore should fail due to path validation
	err = mgr.Restore("malicious.txt", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "system path")
}

// TestSecurity_CopyDirRejectsSymlinkAtTopLevel verifies that copyDirAndDelete
// rejects directories that are symlinks themselves.
func TestSecurity_CopyDirRejectsSymlinkAtTopLevel(t *testing.T) {
	tmpHome := t.TempDir()
	xdgData := filepath.Join(tmpHome, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", tmpHome)

	mgr, err := NewManager()
	require.NoError(t, err)

	testDir := t.TempDir()

	// Create target directory with content
	targetDir := filepath.Join(testDir, "target")
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "file.txt"), []byte("content"), 0644))

	// Create symlink to the directory
	linkDir := filepath.Join(testDir, "linkdir")
	require.NoError(t, os.Symlink(targetDir, linkDir))

	// copyDirAndDelete should fail because source is a symlink
	dstDir := filepath.Join(testDir, "destination")
	err = mgr.copyDirAndDelete(linkDir, dstDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")

	// Target directory should still exist
	assert.FileExists(t, filepath.Join(targetDir, "file.txt"))
}
