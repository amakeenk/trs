package trash

import (
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
