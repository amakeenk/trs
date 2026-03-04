# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/amakeenk/trs/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/amakeenk/trs/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/amakeenk/trs/releases/tag/v0.1.0
