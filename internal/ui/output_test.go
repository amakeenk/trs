package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoColor(t *testing.T) {
	tests := []struct {
		name     string
		noColor  string
		expected bool
	}{
		{
			name:     "NO_COLOR not set",
			noColor:  "",
			expected: false,
		},
		{
			name:     "NO_COLOR set to 1",
			noColor:  "1",
			expected: true,
		},
		{
			name:     "NO_COLOR set to any value",
			noColor:  "anything",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tt.noColor)
			// Reset the environment for this test
			result := noColor()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestColor(t *testing.T) {
	tests := []struct {
		name     string
		noColor  string
		color    string
		text     string
		expected string
	}{
		{
			name:     "with color enabled",
			noColor:  "",
			color:    Red,
			text:     "error",
			expected: "\x1b[31merror\x1b[0m",
		},
		{
			name:     "with NO_COLOR set",
			noColor:  "1",
			color:    Red,
			text:     "error",
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tt.noColor)
			result := Color(tt.color, tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestError(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	result := Error("something went wrong")
	assert.Contains(t, result, "something went wrong")
	assert.Contains(t, result, Red)
}

func TestSuccess(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	result := Success("operation completed")
	assert.Contains(t, result, "operation completed")
	assert.Contains(t, result, Green)
}

func TestWarning(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	result := Warning("be careful")
	assert.Contains(t, result, "be careful")
	assert.Contains(t, result, Yellow)
}

func TestDirectory(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	result := Directory("mydir")
	assert.Contains(t, result, "mydir")
	assert.Contains(t, result, Blue)
}

func TestBoldText(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	result := BoldText("HEADER")
	assert.Contains(t, result, "HEADER")
	assert.Contains(t, result, Bold)
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    3 * 1024 * 1024,
			expected: "3.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    5 * 1024 * 1024 * 1024,
			expected: "5.0 GB",
		},
		{
			name:     "terabytes",
			bytes:    2 * 1024 * 1024 * 1024 * 1024,
			expected: "2.0 TB",
		},
		{
			name:     "fractional kilobytes",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "zero",
			bytes:    0,
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSizeShort(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2.0K",
		},
		{
			name:     "megabytes",
			bytes:    3 * 1024 * 1024,
			expected: "3.0M",
		},
		{
			name:     "gigabytes",
			bytes:    5 * 1024 * 1024 * 1024,
			expected: "5.0G",
		},
		{
			name:     "fractional",
			bytes:    1536,
			expected: "1.5K",
		},
		{
			name:     "zero",
			bytes:    0,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSizeShort(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "maxLen 3",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen 0",
			input:    "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "maxLen 1",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "maxLen 2",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "no padding needed",
			input:    "hello",
			length:   5,
			expected: "hello",
		},
		{
			name:     "padding needed",
			input:    "hi",
			length:   5,
			expected: "hi   ",
		},
		{
			name:     "longer than length",
			input:    "hello world",
			length:   5,
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			length:   3,
			expected: "   ",
		},
		{
			name:     "zero length",
			input:    "hello",
			length:   0,
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadRight(tt.input, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTerminal(t *testing.T) {
	// This test just verifies the function doesn't panic
	// The actual result depends on the execution environment
	result := IsTerminal()
	// In a test environment, stdout is typically not a terminal
	// But we can't assert a specific value since it depends on how tests are run
	_ = result
}
