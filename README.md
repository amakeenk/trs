# trs

**trs** (from **TR**a**S**h) — a secure CLI utility for moving files to trash using the XDG Trash specification.

## Why trs?

- **Safe by default** - Files go to trash, not /dev/null
- **XDG compliant** - Works with GNOME, KDE, and other desktop environments
- **Security-focused** - Protected against symlink attacks, path traversal, and DoS
- **No dependencies** - Pure Go, no external utilities needed
- **Cross-device** - Handles files on different filesystems automatically

## Installation

### ALT Linux Sisyphus

```bash
# apt-get install trs
```

### Go Install

```bash
go install altlinux.space/amakeenk/trs@latest
```

## Usage

### Move files to trash

```bash
trs file.txt              # Move a file to trash
trs file1.txt file2.txt   # Move multiple files
trs -r directory/         # Move a directory (recursive)
trs -f nonexistent        # Force (ignore nonexistent files)
trs -v file.txt           # Verbose output
```

### List trashed files

```bash
trs list                  # List all trashed files
trs list --json           # JSON output
```

Output:
```
#  NAME              SIZE    DELETED              ORIGINAL PATH
1  config.json       1.2KB   2026-03-05 10:30    /home/user/project/config.json
2  old_backup/       45MB    2026-03-04 18:22    /home/user/backups/old_backup
```

### Manage trash files

```bash
trs manage              # Interactive TUI with fuzzy search
trs manage config.json  # Restore by name
trs manage 1            # Restore by index (from list)
trs manage --last       # Restore most recently trashed file
trs manage -f file.txt  # Force overwrite existing file
trs manage --json       # JSON output
```

TUI features:
- Multi-select with Tab/Space, arrow navigation
- Fuzzy search across names and original paths
- Press `r` to restore or `d` to permanently delete selected files
- Results show full original paths

### Empty trash

```bash
trs empty                # Empty entire trash
trs empty --days 7       # Only files older than 7 days
trs empty --json         # JSON output
```

### Check trash status

```bash
trs status               # Show trash statistics
trs status -v            # Verbose (oldest/newest/largest files)
trs status --json        # JSON output
```

Output:
```
Trash: 42 files, 128.5 MB
```

With `-v`:
```
Trash: 42 files, 128.5 MB
Oldest: 2026-02-01 10:00  Newest: 2026-03-10 15:30
Largest: backup.tar (50MB), video.mp4 (25MB), data.zip (10MB)
```

### Version

```bash
trs version
trs version --json
```

### Shell completion

```bash
trs completion bash      # Bash
trs completion zsh       # Zsh
trs completion fish      # Fish
trs completion powershell # PowerShell
```

## Features

### XDG Trash Specification

- Home trash: `~/.local/share/Trash/`
- Volume trash: `$VOLUME/.Trash-$UID/`
- Cross-device moves handled automatically
- Compatible with desktop environment trash

### Interactive TUI

The `trs manage` command launches an interactive terminal UI:
- Fuzzy search across file names and original paths
- Arrow key navigation with live preview
- Multi-select files with Tab/Space
- Press `r` to restore or `d` to permanently delete selected files
- Confirmation screen before action
- Works in any terminal

### Security

trs implements multiple security measures:

- **Symlink protection** - Never follows symlinks when copying or removing
- **Path validation** - Prevents path traversal attacks
- **TrashInfo validation** - Validates paths from `.trashinfo` files
- **DoS prevention** - Size and iteration limits on all operations
- **Directory validation** - Verifies trash directory ownership and permissions

### JSON Output

All commands support `--json` for scripting:

```bash
trs list --json | jq '.[] | .name'
trs status --json | jq '.totalSize'
```

## Configuration

### Environment Variables

- `XDG_DATA_HOME` - Custom data directory (default: `~/.local/share`)
- `NO_COLOR` - Disable colored output

### Exit Codes

- `0` - Success
- `1` - Error
- `2` - Invalid usage

## Comparison with `rm`

| Feature | `rm` | `trs` |
|---------|------|-------|
| Permanent deletion | Yes | No |
| Trash can | No | Yes |
| Undo/restore | No | Yes |
| Safe by default | No | Yes |
| XDG compliant | No | Yes |

## Development

### Build

```bash
make build           # Build binary
make test            # Run tests
make coverage        # Test coverage
make coverage-html   # Coverage HTML report
make install         # Install to $GOPATH/bin
```

### Test

```bash
go test ./... -v
go test -run TestManager_Move ./internal/trash/...
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## Note

This project was entirely implemented with AI assistance.
