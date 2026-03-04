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

// Truncate truncates a string to maxLen, adding "..." if needed
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// PadRight pads a string to a given length with spaces
func PadRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// IsTerminal checks if stdout is a terminal
func IsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
