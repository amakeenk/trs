//go:build !linux

package trash

import (
	"os"
)

func renameNoReplace(oldpath, newpath string) error {
	// Fallback for non-Linux: vulnerable to TOCTOU but best effort
	if _, err := os.Lstat(newpath); err == nil {
		return os.ErrExist
	}
	return os.Rename(oldpath, newpath)
}
