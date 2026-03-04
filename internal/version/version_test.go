package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	info := Get()

	assert.Equal(t, Version, info.Version)
	assert.Equal(t, GitCommit, info.GitCommit)
	assert.Equal(t, BuildDate, info.BuildDate)
	assert.Equal(t, GoVersion, info.GoVersion)
	assert.NotEmpty(t, info.Platform)
	assert.Contains(t, info.Platform, "/")
}

func TestString(t *testing.T) {
	result := String()

	assert.Contains(t, result, "trs version")
	assert.Contains(t, result, Version)
	assert.Contains(t, result, "Git commit:")
	assert.Contains(t, result, GitCommit)
	assert.Contains(t, result, "Build date:")
	assert.Contains(t, result, BuildDate)
	assert.Contains(t, result, "Go version:")
	assert.Contains(t, result, GoVersion)
	assert.Contains(t, result, "Platform:")
}

func TestDefaultValues(t *testing.T) {
	// Test that default values are set correctly
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "unknown", GitCommit)
	assert.Equal(t, "unknown", BuildDate)
}

func TestInfoPlatform(t *testing.T) {
	info := Get()

	// Platform should be in format "os/arch"
	parts := strings.Split(info.Platform, "/")
	assert.Len(t, parts, 2)
	assert.NotEmpty(t, parts[0]) // OS
	assert.NotEmpty(t, parts[1]) // Arch
}
