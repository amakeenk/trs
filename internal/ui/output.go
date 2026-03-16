package ui

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	Reset  = "\x1b[0m"
	Red    = "\x1b[31m"
	Green  = "\x1b[32m"
	Yellow = "\x1b[33m"
	Blue   = "\x1b[34m"
	Bold   = "\x1b[1m"
)

// noColor checks if colors should be disabled
func noColor() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	// Check if stdout is not a terminal
	// For simplicity, we always use colors unless NO_COLOR is set
	return false
}

// Color wraps text in ANSI color codes
func Color(color, text string) string {
	if noColor() {
		return text
	}
	return color + text + Reset
}

// Error formats an error message in red
func Error(msg string) string {
	return Color(Red, msg)
}

// Success formats a success message in green
func Success(msg string) string {
	return Color(Green, msg)
}

// Warning formats a warning message in yellow
func Warning(msg string) string {
	return Color(Yellow, msg)
}

// Directory formats a directory name (blue with trailing /)
func Directory(name string) string {
	return Color(Blue, name)
}

// BoldText wraps text in bold
func BoldText(text string) string {
	return Color(Bold, text)
}

// FormatSize formats bytes into human-readable size
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSizeShort formats bytes into short human-readable size (without space)
func FormatSizeShort(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d", bytes)
	}
}

func DisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		width++
		if r >= 0x1100 && (r <= 0x115F ||
			r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0x3247 && r != 0x303F) ||
			(r >= 0x3250 && r <= 0x4DBF) ||
			(r >= 0x4E00 && r <= 0xA4C6) ||
			(r >= 0xA960 && r <= 0xA97C) ||
			(r >= 0xAC00 && r <= 0xD7A3) ||
			(r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0xFE10 && r <= 0xFE1F) ||
			(r >= 0xFE30 && r <= 0xFE6B) ||
			(r >= 0xFF01 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x1B000 && r <= 0x1B001) ||
			(r >= 0x1F200 && r <= 0x1F251) ||
			(r >= 0x20000 && r <= 0x3FFFD)) {
			width++
		}
	}
	return width
}

func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if DisplayWidth(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	width := 0
	for i, r := range runes {
		rw := 1
		if r >= 0x1100 {
			rw = 2
		}
		if width+rw > maxLen-3 {
			return string(runes[:i]) + "..."
		}
		width += rw
	}
	return s
}

func PadRight(s string, length int) string {
	dw := DisplayWidth(s)
	if dw >= length {
		return s
	}
	return s + strings.Repeat(" ", length-dw)
}

// IsTerminal checks if stdout is a terminal
func IsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
