package trash

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TrashInfo represents the content of a .trashinfo file
type TrashInfo struct {
	Path         string    // Original absolute path
	DeletionDate time.Time // When the file was trashed
}

// TimeFormat is the XDG Trash spec time format
const TimeFormat = "2006-01-02T15:04:05"

// ParseTrashInfo reads and parses a .trashinfo file
func ParseTrashInfo(path string) (*TrashInfo, error) {
	// Limit file size to prevent DoS
	const maxTrashInfoSize = 8192 // 8KB is more than enough for valid trashinfo

	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat trashinfo %s: %w", path, err)
	}
	if stat.Size() > maxTrashInfoSize {
		return nil, fmt.Errorf("trashinfo file too large: %d bytes (max %d)", stat.Size(), maxTrashInfoSize)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open trashinfo %s: %w", path, err)
	}
	defer f.Close()

	// Use io.LimitReader as extra safety
	return parseTrashInfoReader(io.LimitReader(f, maxTrashInfoSize))
}

// ParseTrashInfoFromRoot reads and parses a .trashinfo file using TrashRoot.
// This is traversal-resistant - the file must be within the trash root.
func ParseTrashInfoFromRoot(root *TrashRoot, name string) (*TrashInfo, error) {
	const maxTrashInfoSize = 8192

	stat, err := root.LstatInfo(name)
	if err != nil {
		return nil, fmt.Errorf("stat trashinfo %s: %w", name, err)
	}
	if stat.Size() > maxTrashInfoSize {
		return nil, fmt.Errorf("trashinfo file too large: %d bytes (max %d)", stat.Size(), maxTrashInfoSize)
	}

	f, err := root.OpenInfo(name)
	if err != nil {
		return nil, fmt.Errorf("open trashinfo %s: %w", name, err)
	}
	defer f.Close()

	return parseTrashInfoReader(io.LimitReader(f, maxTrashInfoSize))
}

func parseTrashInfoReader(r io.Reader) (*TrashInfo, error) {
	info := &TrashInfo{}
	scanner := bufio.NewScanner(r)

	// Limit line length to prevent memory exhaustion
	const maxLineLen = 8192
	scanner.Buffer(make([]byte, maxLineLen), maxLineLen)

	// Skip [Trash Info] header
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty trashinfo file")
	}
	if strings.TrimSpace(scanner.Text()) != "[Trash Info]" {
		return nil, fmt.Errorf("invalid trashinfo: missing [Trash Info] header")
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

switch key {
		case "Path":
			// URL decode the path
			decoded, err := url.QueryUnescape(value)
			if err != nil {
				return nil, fmt.Errorf("invalid path encoding: %w", err)
			}

			// Validate the path
			if !filepath.IsAbs(decoded) {
				return nil, fmt.Errorf("trashinfo path must be absolute: %s", decoded)
			}

			// Check for suspicious patterns
			if strings.Contains(decoded, "\x00") {
				return nil, fmt.Errorf("trashinfo path contains null byte")
			}

			// Limit path length
			if len(decoded) > 4096 {
				return nil, fmt.Errorf("trashinfo path too long")
			}

			info.Path = decoded
case "DeletionDate":
			t, err := time.Parse(TimeFormat, value)
			if err != nil {
				return nil, fmt.Errorf("parse deletion date: %w", err)
			}
			info.DeletionDate = t
		}
	}

	if info.Path == "" {
		return nil, fmt.Errorf("trashinfo missing Path")
	}

	return info, scanner.Err()
}

// Write writes the TrashInfo to a .trashinfo file
func (ti *TrashInfo) Write(path string) error {
	var buf strings.Builder
	buf.WriteString("[Trash Info]\n")
	buf.WriteString(fmt.Sprintf("Path=%s\n", url.QueryEscape(ti.Path)))
	buf.WriteString(fmt.Sprintf("DeletionDate=%s\n", ti.DeletionDate.Format(TimeFormat)))

return os.WriteFile(path, []byte(buf.String()), 0600)
}

// WriteExclusive writes the TrashInfo to a .trashinfo file atomically using O_EXCL
// Returns os.ErrExist if the file already exists
func (ti *TrashInfo) WriteExclusive(path string) error {
	var buf strings.Builder
	buf.WriteString("[Trash Info]\n")
	buf.WriteString(fmt.Sprintf("Path=%s\n", url.QueryEscape(ti.Path)))
	buf.WriteString(fmt.Sprintf("DeletionDate=%s\n", ti.DeletionDate.Format(TimeFormat)))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(buf.String())
	return err
}

// TrashInfoPath returns the path to the .trashinfo file for a given trash name
func TrashInfoPath(trashDir, name string) string {
	return filepath.Join(trashDir, "info", name+".trashinfo")
}

// FilesPath returns the path where the trashed file should be stored
func FilesPath(trashDir, name string) string {
	return filepath.Join(trashDir, "files", name)
}
