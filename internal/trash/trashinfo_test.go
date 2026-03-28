package trash

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrashInfo(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *TrashInfo
		wantErr bool
	}{
		{
			name: "valid basic",
			input: `[Trash Info]
Path=/home/user/file.txt
DeletionDate=2024-01-15T10:30:00`,
			want: &TrashInfo{
				Path:         "/home/user/file.txt",
				DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
		{
			name: "path with spaces",
			input: `[Trash Info]
Path=/home/user/my%20file.txt
DeletionDate=2024-01-15T10:30:00`,
			want: &TrashInfo{
				Path:         "/home/user/my file.txt",
				DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
		{
			name: "path with special chars",
			input: `[Trash Info]
Path=/home/user/%D1%84%D0%B0%D0%B9%D0%BB.txt
DeletionDate=2024-01-15T10:30:00`,
			want: &TrashInfo{
				Path:         "/home/user/файл.txt",
				DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
		{
			name: "missing header",
			input: `Path=/home/user/file.txt
DeletionDate=2024-01-15T10:30:00`,
			wantErr: true,
		},
		{
			name: "missing path",
			input: `[Trash Info]
DeletionDate=2024-01-15T10:30:00`,
			wantErr: true,
		},
		{
			name: "invalid date format",
			input: `[Trash Info]
Path=/home/user/file.txt
DeletionDate=2024/01/15 10:30`,
			wantErr: true,
		},
		{
			name:    "empty file",
			input:   ``,
			wantErr: true,
		},
		{
			name: "with extra whitespace",
			input: `[Trash Info]
  Path = /home/user/file.txt  
  DeletionDate = 2024-01-15T10:30:00  
`,
			want: &TrashInfo{
				Path:         "/home/user/file.txt",
				DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTrashInfoReader(strings.NewReader(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Path, got.Path)
			assert.Equal(t, tt.want.DeletionDate, got.DeletionDate)
		})
	}
}

func TestTrashInfoWrite(t *testing.T) {
	ti := &TrashInfo{
		Path:         "/home/user/file.txt",
		DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	path := t.TempDir() + "/test.trashinfo"
	err := ti.Write(path)
	require.NoError(t, err)

	// Read back and verify
	got, err := ParseTrashInfo(path)
	require.NoError(t, err)
	assert.Equal(t, ti.Path, got.Path)
	assert.Equal(t, ti.DeletionDate, got.DeletionDate)
}

func TestTrashInfoWriteWithSpecialChars(t *testing.T) {
	ti := &TrashInfo{
		Path:         "/home/user/файл с пробелами.txt",
		DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	path := t.TempDir() + "/test.trashinfo"
	err := ti.Write(path)
	require.NoError(t, err)

	// Read back and verify
	got, err := ParseTrashInfo(path)
	require.NoError(t, err)
	assert.Equal(t, ti.Path, got.Path)
}

func TestTrashInfoPath(t *testing.T) {
	got := TrashInfoPath("/home/user/.local/share/Trash", "file.txt")
	assert.Equal(t, "/home/user/.local/share/Trash/info/file.txt.trashinfo", got)
}

func TestFilesPath(t *testing.T) {
	got := FilesPath("/home/user/.local/share/Trash", "file.txt")
	assert.Equal(t, "/home/user/.local/share/Trash/files/file.txt", got)
}

// TestParseTrashInfo_Symlink verifies that ParseTrashInfo uses os.Lstat (not os.Stat)
// for size check. With os.Stat, a symlink to a large file would fail the size check.
// With os.Lstat, we check the symlink size itself (tiny), not the target.
func TestParseTrashInfo_Symlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid trashinfo file larger than 8192 bytes (the size limit)
	// We'll create a path that's long enough to exceed the limit
	longPath := "/home/user/" + strings.Repeat("a", 8200) + ".txt"
	ti := &TrashInfo{
		Path:         longPath,
		DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	// Write the large trashinfo file
	largeInfoPath := tmpDir + "/large.trashinfo"
	err := ti.Write(largeInfoPath)
	require.NoError(t, err)

	// Create a symlink to the large file
	symlinkPath := tmpDir + "/symlink.trashinfo"
	err = os.Symlink(largeInfoPath, symlinkPath)
	require.NoError(t, err)

	// With os.Stat (wrong): fails because target file > 8192 bytes
	// With os.Lstat (correct): passes because symlink itself is tiny
	//
	// We expect ParseTrashInfo to proceed past size check (using Lstat)
	// and then fail on parsing the content (it's valid content, just large)
	//
	// Actually: The content IS valid, just the path is very long.
	// The parser should handle it since io.LimitReader will still read it.
	// Let's check what happens...
	got, err := ParseTrashInfo(symlinkPath)

	// The key assertion: we should NOT get an error about file size
	// If we get "trashinfo file too large", it means os.Stat is being used (wrong)
	if err != nil && strings.Contains(err.Error(), "file too large") {
		t.Fatalf("ParseTrashInfo used os.Stat (follows symlinks): %v", err)
	}

	// With os.Lstat, the symlink itself is small, so we proceed to parsing
	// The actual parsing should work (content is valid, just has long path)
	// But wait - the file content is > 8192, so io.LimitReader will truncate it
	// This means parsing will likely fail due to incomplete content
	//
	// The important thing is: we should NOT fail on the initial size check
	_ = got // May be nil if parsing fails due to truncation, that's OK
}

func TestDeletionDateWithTimezone(t *testing.T) {
	input := `[Trash Info]
Path=/home/user/file.txt
DeletionDate=2024-01-15T10:30:00Z`
	got, err := parseTrashInfoReader(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), got.DeletionDate)

	input = `[Trash Info]
Path=/home/user/file.txt
DeletionDate=2024-01-15T10:30:00+03:00`
	got, err = parseTrashInfoReader(strings.NewReader(input))
	require.NoError(t, err)
	_, expectedOffset := time.Date(2024, 1, 15, 10, 30, 0, 0, time.FixedZone("", 3*3600)).Zone()
	_, actualOffset := got.DeletionDate.Zone()
	assert.Equal(t, expectedOffset, actualOffset)
	assert.True(t, got.DeletionDate.Equal(time.Date(2024, 1, 15, 10, 30, 0, 0, time.FixedZone("", 3*3600))))

	input = `[Trash Info]
Path=/home/user/file.txt
DeletionDate=2024-01-15T10:30:00`
	got, err = parseTrashInfoReader(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), got.DeletionDate)

	ti := &TrashInfo{
		Path:         "/home/user/file.txt",
		DeletionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	path := t.TempDir() + "/test.trashinfo"
	err = ti.Write(path)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "DeletionDate=2024-01-15T10:30:00Z")
}
