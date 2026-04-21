# ghpr-tui

TUI for reviewing GitHub Pull Requests locally, built with Go + Bubbletea.

## Version

- Version is defined in `internal/version/version.go` as `v{n}` (e.g. `v7`).
- **Every commit that changes behavior or adds features MUST increment the version number.** Bump the integer by 1.
- This includes bug fixes, new features, UI changes, keybind changes — anything user-facing.
- Do NOT bump for docs-only or CLAUDE.md-only changes.

## Build & Install

```sh
go build -o ghpr-tui .
go install .
```

## Project Structure

- `main.go` — Entry point, arg parsing, --version/--help
- `internal/version/version.go` — Version constant (bump on every change)
- `internal/app/app.go` — Top-level Bubbletea model, screen routing
- `internal/ghclient/` — GitHub CLI wrapper (`gh` subprocess calls), diff parser
- `internal/state/` — Persistent review state (`~/.config/ghpr-tui/state.json`)
- `internal/ui/prlist/` — PR inbox list screen
- `internal/ui/filelist/` — Changed files list screen
- `internal/ui/diffview/` — Single file diff viewer screen
- `internal/ui/checks/` — CI checks detail screen
- `internal/ui/styles/` — Shared lipgloss styles
