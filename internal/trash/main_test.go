package trash

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Isolate tests from host mount points by default
	origMounts := mountsFilePath
	mountsFilePath = "/dev/null"

	code := m.Run()

	mountsFilePath = origMounts
	os.Exit(code)
}
