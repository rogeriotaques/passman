# Commands

## Overview

Commands are grouped by noun. All commands use `--vault <name>` to target a
specific vault (default: "default").

| Command | File | Description |
|---------|------|-------------|
| `secret add` | `cmd/add.go` | Add a secret (name + value) |
| `secret get` | `cmd/get.go` | Retrieve a secret (clipboard or stdout) |
| `secret list` | `cmd/list.go` | List/search entries with interactive selector |
| `secret remove` | `cmd/remove.go` | Remove a secret by name |
| `secret export` | `cmd/export.go` | Export secrets as env vars or inject into subprocess |
| `secret import` | `cmd/import.go` | Import from .env or .csv files |
| `vault init` | `cmd/init.go` | Create a new encrypted vault |
| `vault list` | `cmd/vault_list.go` | List all vault names |
| `vault config` | `cmd/config.go` | View/set vault configuration |
| `vault sync` | `cmd/sync.go` | Git sync (commit + pull + push) |
| `vault passwd` | `cmd/passwd.go` | Change master password |
| `vault destroy` | `cmd/destroy.go` | Permanently delete a vault or all vaults |
| `generate` | `cmd/generate.go` | Generate a random password |
| `lock` | `cmd/lock.go` | Clear agent cached passwords |
| `upgrade` | `cmd/upgrade.go` | Upgrade to the latest version |

The groups are defined in `cmd/secret.go` and `cmd/vault.go`; the hidden
`_agent` daemon stays a root command.

## DRY Helpers

- `app.loadVault()` — unlock vault + load entries (used by get/list/add/remove/export/import)
- `app.saveAndSync()` — save vault + trigger auto-sync (used by add/remove/import/passwd)
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
