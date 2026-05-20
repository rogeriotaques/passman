# Commands

## Overview

All commands use `--vault <name>` to target a specific vault (default: "default").

| Command | File | Description |
|---------|------|-------------|
| `init` | `cmd/init.go` | Create a new encrypted vault |
| `add` | `cmd/add.go` | Add a secret (name + value) |
| `get` | `cmd/get.go` | Retrieve a secret (clipboard or stdout) |
| `list` | `cmd/list.go` | List/search entries with interactive selector |
| `rm` | `cmd/rm.go` | Remove a secret by name |
| `generate` | `cmd/generate.go` | Generate a random password |
| `export` | `cmd/export.go` | Export secrets as env vars or inject into subprocess |
| `import` | `cmd/import.go` | Import from .env or .csv files |
| `config` | `cmd/config.go` | View/set vault configuration |
| `sync` | `cmd/sync.go` | Git sync (commit + pull + push) |
| `passwd` | `cmd/passwd.go` | Change master password |
| `lock` | `cmd/lock.go` | Clear agent cached passwords |
| `vaults` | `cmd/vaults.go` | List all vault names |
| `destroy` | `cmd/destroy.go` | Permanently delete a vault or all vaults |

## DRY Helpers

- `app.loadVault()` — unlock vault + load entries (used by get/list/add/rm/export/import)
- `app.saveAndSync()` — save vault + trigger auto-sync (used by add/rm/import/passwd)
- `copyToClipboard()` — copy to clipboard with auto-clear message
- `toEnvVar()` — convert entry name to SCREAMING_SNAKE_CASE
- `shellEscape()` — escape single quotes for shell safety

## Entry Model

```go
type Entry struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}
```

Simple key-value. Name is case-insensitive for lookup. Value is the secret.

## Config Keys

| Key | Values | Default | Description |
|-----|--------|---------|-------------|
| `auto-sync` | `on`/`off` | `off` | Auto git sync after writes |
| `session-timeout` | minutes | `15` | Agent cache TTL (0 disables) |
| `git` | URL | empty | Git remote URL for the vault |
