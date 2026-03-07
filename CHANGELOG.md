# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://altlinux.space/amakeenk/trs/compare/v0.4.1...HEAD
[0.4.1]: https://altlinux.space/amakeenk/trs/compare/v0.4.0...v0.4.1
[0.4.0]: https://altlinux.space/amakeenk/trs/compare/v0.3.0...v0.4.0
[0.3.0]: https://altlinux.space/amakeenk/trs/compare/v0.2.0...v0.3.0
[0.2.0]: https://altlinux.space/amakeenk/trs/compare/v0.1.0...v0.2.0
[0.1.0]: https://altlinux.space/amakeenk/trs/releases/tag/v0.1.0
