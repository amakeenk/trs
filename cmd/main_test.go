package cmd

import (
	"os"
	"testing"

	"altlinux.space/amakeenk/trs/internal/trash"
)

func TestMain(m *testing.M) {
	// Isolate tests from host mount points
	trash.SetMountsFilePath("/dev/null")

	code := m.Run()

	os.Exit(code)
}
