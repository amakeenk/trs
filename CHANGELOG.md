# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.10.0] - 2026-04-25

### Added
- Support XDG spec for `.Trash/$UID` volume directories
- ORIGINAL column in `trs list` output

### Fixed
- Resolve TOCTOU race condition during file restoration
- Recreate symlinks across devices instead of failing
- Correctly handle unicode characters in TUI search
- Prevent `r` and `d` hotkeys from interrupting search input in TUI
- Ignore non-existent files correctly with `--force` flag
- Allow trashing files on network filesystems with mapped UIDs

## [0.9.0] - 2026-04-03

### Added
- "Select all" (`a`, `Ctrl+a`) and "Deselect all" (`A`) features in management TUI
- Support for selecting all filtered items in search mode

### Changed
- Improved UX in TUI: allow quick restore/delete actions on highlighted items without explicit selection
- Updated documentation to reflect correct command names and new features

### Fixed
- Improved robustness of file removal and cleanup operations
- Prevented potential goroutine leaks in filesystem operations
- Fixed cross-device moves for directories containing symlinks

### Security
- Prevented TOCTOU (Time-of-Check to Time-of-Use) attacks on trash directory creation and ownership verification
- Prevented TOCTOU attacks during recursive removal of trashed items
- Enabled safe recursive removal that protects against symlink attacks using modern Go/Unix primitives

## [0.8.0] - 2026-03-28

### Fixed
- Limit input size in interactive prompt for empty command
- Correct defer cleanup in `copyFile` to prevent resource leaks
- Reject negative `--days` at CLI level in empty command
- Use RFC3339 for DeletionDate with timezone in trashinfo
- Reject negative days in `EmptyOlderThan`
- Prevent `GetAllTrashDirs` from blocking on unresponsive mounts

### Security
- Input size limiting prevents DoS via oversized interactive input
- Rejecting negative values prevents unexpected behavior in age-based operations

## [0.7.0] - 2026-03-18

### Added
- Manage command with delete functionality - permanently delete files from trash via TUI
- Delete method in trash manager for permanent file removal

### Changed
- Show full original path in TUI results view
- Rename restore.go to manage.go to reflect expanded functionality

### Fixed
- Exit silently when user cancels in TUI results mode

### Security
- Added tests for Delete method covering directory and not found cases

## [0.6.0] - 2026-03-16

### Added
- Fixed-width table columns with proper Unicode support in `trs list`
- Tab key for file selection in restore TUI
- Multi-select with search in restore TUI
- Improved TUI navigation and display for restore command
- Show individual files with types in verbose mode

### Fixed
- Calculate directory size including contents (was showing 0 for directories)

## [0.5.0] - 2026-03-08

### Fixed
- Handle read-only files and symlinks in empty command
- Add input validation to CLI prompts

### Security
- Use os.Root for traversal-resistant trash operations (prevents symlink attacks)



## [0.4.2] - 2026-03-07

### Added
- Shell completion support for bash, zsh, fish, and powershell via `trs completion` command
- File completion for trs command arguments

## [0.4.1] - 2026-03-07

### Fixed
- Allow /usr/src paths for restore validation (fixes RPM build test failures)

### Added
- Man page (trs.1) with install-man/uninstall-man Makefile targets

## [0.4.0] - 2026-03-07

### Fixed
- Add chmod hint to insecure permissions error message
- Aggregate trash from all directories (home trash + volume-specific .Trash-$UID)

### Changed
- Remove unused --rm flag (was declared but non-functional)

## [0.3.0] - 2026-03-05

### Security

**Critical fixes:**
- Fix symlink following vulnerability in `copyFile()` - now uses `Lstat()` and rejects symlinks to prevent writing to arbitrary files
- Fix symlink attack in `os.RemoveAll()` during restore - implemented `safeRemoveAll()` that doesn't follow symlinks in subdirectories

**High severity fixes:**
- Fix path traversal validation bypass in `validateRestorePath()` - now uses `filepath.Rel()` for robust validation
- Fix TrashInfo path injection - validate paths from `.trashinfo` files (absolute path required, null byte check, length limit)

**Medium severity fixes:**
- Fix DoS via infinite loop in `resolveNameConflict()` - added 10000 iteration limit
- Fix unbounded TrashInfo file size - limit to 8KB to prevent memory exhaustion
- Fix missing trash directory validation - verify ownership, permissions (0700), and reject symlinks

### Changed
- Trash directories are now validated for secure permissions and ownership
- All `.trashinfo` file parsing now enforces size limits and path validation

## [0.2.0] - 2026-03-04

## [0.2.0] - 2026-03-04

### Added
- Interactive TUI for `trs restore` command with fuzzy search filtering
- Real-time search across file names and original paths
- Arrow key navigation with item preview
- Non-TTY fallback for scripted environments
- AGENTS.md documentation for coding assistants

### Dependencies
- github.com/charmbracelet/bubbletea - TUI framework
- github.com/charmbracelet/bubbles - TUI components
- github.com/charmbracelet/lipgloss - Styling

## [0.1.0] - 2026-03-03

### Added
- Initial release
- `trs <files...>` - Move files to trash (supports `-r`, `-f`, `-v` flags)
- `trs list` - List trashed files with index, size, date
- `trs restore [name|index]` - Restore files from trash
- `trs restore --last` - Restore most recently trashed file
- `trs empty [--days N]` - Clear trash (optional age filter)
- `trs status [-v]` - Show trash statistics
- `trs version` - Display version information
- XDG Trash specification compliance (no external utilities)
- Cross-device moves via copy + delete fallback
- Name conflict resolution with `_1`, `_2` suffixes
- Volume-specific trash (`$VOLUME/.Trash-$UID/`)
- Symlink handling (trash link, not target)
- JSON output support (`--json`)
- ANSI color output with `NO_COLOR` support
- 80.3% test coverage

### Dependencies
- github.com/spf13/cobra - CLI framework
- github.com/stretchr/testify - Testing assertions

[Unreleased]: https://altlinux.space/amakeenk/trs/compare/v0.10.0...HEAD
[0.10.0]: https://altlinux.space/amakeenk/trs/compare/v0.9.0...v0.10.0
[0.9.0]: https://altlinux.space/amakeenk/trs/compare/v0.8.0...v0.9.0
[0.8.0]: https://altlinux.space/amakeenk/trs/compare/v0.7.0...v0.8.0
[0.7.0]: https://altlinux.space/amakeenk/trs/compare/v0.6.0...v0.7.0
[0.6.0]: https://altlinux.space/amakeenk/trs/compare/v0.5.0...v0.6.0
[0.5.0]: https://altlinux.space/amakeenk/trs/compare/v0.4.2...v0.5.0
[0.4.2]: https://altlinux.space/amakeenk/trs/compare/v0.4.1...v0.4.2
[0.4.1]: https://altlinux.space/amakeenk/trs/compare/v0.4.0...v0.4.1
[0.4.0]: https://altlinux.space/amakeenk/trs/compare/v0.3.0...v0.4.0
[0.3.0]: https://altlinux.space/amakeenk/trs/compare/v0.2.0...v0.3.0
[0.2.0]: https://altlinux.space/amakeenk/trs/compare/v0.1.0...v0.2.0
[0.1.0]: https://altlinux.space/amakeenk/trs/releases/tag/v0.1.0
