# AGENTS.md - trs Codebase Guide

CLI utility for safe file deletion using XDG Trash specification.

## Build Commands

```bash
# Build binary
make build
# or: go build -o trs .

# Build with version injection
make VERSION=v1.0.0 build

# Run tests (all)
make test
go test ./... -v

# Run single test
go test -run TestManager_Move ./internal/trash/...
go test -run TestParseTrashInfo ./internal/trash/...

# Run tests with coverage
make coverage
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out

# Coverage HTML report
make coverage-html

# Install to $GOPATH/bin
make install

# Cross-compile all platforms
make build-all
```

## Project Structure

```
trs/
├── main.go              # Entry point (minimal)
├── cmd/                 # Cobra CLI commands
│   ├── root.go         # Root command, global flags
│   ├── trash.go        # trs <files...>
│   ├── list.go         # trs list
│   ├── restore.go      # trs restore [--last]
│   ├── empty.go        # trs empty [--days N]
│   ├── status.go       # trs status [-v]
│   └── version.go      # trs version
├── internal/
│   ├── trash/          # Core trash logic
│   │   ├── manager.go     # TrashManager, all operations
│   │   ├── trashinfo.go   # .trashinfo file handling
│   │   └── volume.go      # Volume/mount detection
│   ├── ui/             # Output formatting
│   │   └── output.go      # Colors, size formatting
│   └── version/        # Version info (ldflags)
│       └── version.go
├── Makefile
└── go.mod
```

## Code Style Guidelines

### Imports

Standard import order (separated by blank lines):
1. Standard library
2. External packages
3. Internal packages

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/amakeenk/trs/internal/trash"
)
```

### Naming Conventions

- **Packages**: lowercase, single word (`trash`, `ui`, `version`)
- **Types**: PascalCase (`TrashInfo`, `TrashItem`, `Manager`)
- **Functions**: PascalCase (exported), camelCase (private)
- **Constants**: PascalCase (`TimeFormat`, `Reset`)

```go
// Public
func NewManager() (*Manager, error) { ... }

// Private
func (m *Manager) resolveNameConflict(...) { ... }
```

### Struct Fields

Always document struct fields with inline comments:

```go
type TrashItem struct {
	Name         string    // Name in trash
	OriginalPath string    // Original absolute path
	DeletionDate time.Time // When it was trashed
	Size         int64     // Size in bytes
	IsDir        bool      // Is directory
}
```

### Error Handling

- Always wrap errors with context using `fmt.Errorf` and `%w`
- Never use `panic()` - return errors
- Provide actionable error messages

```go
// Good
if err != nil {
	return fmt.Errorf("get absolute path: %w", err)
}
return fmt.Errorf("cannot trash '%s': No such file or directory", path)

// Bad
if err != nil {
	return err
}
if err != nil {
	panic(err)
}
```

### Exit Codes

- `0` - Success, `1` - Error, `2` - Invalid usage

```go
os.Exit(1)  // Error
os.Exit(2)  // Invalid usage
```

## Testing Guidelines

### Test Structure

Use table-driven tests for multiple cases. Example:

```go
func TestParseTrashInfo(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *TrashInfo
		wantErr bool
	}{
		{ name: "valid", input: `...`, want: &TrashInfo{...} },
		{ name: "invalid", input: `...`, wantErr: true },
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
		})
	}
}
```

### Assertions

- Use `require` for setup assertions (must pass)
- Use `assert` for test assertions (can continue)

```go
require.NoError(t, err)  // Setup - must succeed
assert.Equal(t, expected, actual)  // Test assertion
```

### Test Environment

Use `t.TempDir()` for temp directories, `t.Setenv()` for env vars:

```go
func TestSomething(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))
	t.Setenv("HOME", tmpHome)
	// ...
}
```

## CLI Patterns

### Global Flags (cmd/root.go)

```go
var (
	flagVerbose bool
	flagJSON    bool
	flagForce   bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "machine-readable JSON output")
}
```

### JSON Output

All list/status commands support `--json`:

```go
if flagJSON {
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}
```

## ANSI Colors (internal/ui/output.go)

```go
ui.Error("message")    // Red
ui.Success("message")  // Green
ui.Warning("message")  // Yellow
ui.Directory("name/")  // Blue (for dir names)
ui.BoldText("HEADER")  // Bold text
// Respect NO_COLOR env var automatically
```

## Key Architecture Decisions

1. **No external utilities** - XDG Trash spec implemented in pure Go
2. **No CGO** - Pure Go, static binary possible
3. **Symlink handling** - Use `os.Lstat()` to operate on link, not target
4. **Cross-device moves** - `os.Rename()` falls back to copy+delete on EXDEV
5. **Name conflicts** - Append `_1`, `_2`, etc. suffix to name in trash
6. **Volume trash** - Use `$VOLUME/.Trash-$UID/` for cross-filesystem files

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/stretchr/testify` - Testing assertions

No other dependencies allowed without strong justification.
