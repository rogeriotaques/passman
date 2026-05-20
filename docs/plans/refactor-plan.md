# Refactor Plan: POC → DOCS.md Spec (v2 — post-review)

## Goal

Strip the POC down to the DOCS.md spec, discarding features not listed,
implementing missing ones, and using TDD throughout. No dead code. DRY helpers.
Security-audit ready.

## Entry Model Change

**Old:** `{ Name, Username, Password, Notes, Tags, TotpSecret, CreatedAt, UpdatedAt }`
**New:** `{ Name, Value string }`

Clean break — bump `EncryptedBlob.Version` to 2. No migration (pre-release).

## Commands: Final Spec

| Command | Behavior |
|---------|----------|
| `init [vault-name]` | Create vault. Password optional (min 8 if provided, warn if empty). `--git` optionally prompts for remote URL. Vault-name defaults to "default". |
| `add <name> [--vault]` | Prompt for value. Error on duplicate. |
| `get <name> [--vault] [--print]` | Clipboard + 30s auto-clear. `--print` outputs to stdout. |
| `list [query...] [--vault] [--all]` | 20 entries default, `--all` shows everything. Multi-token search on names. Interactive selector ALWAYS active in TTY (arrow keys + ENTER retrieves). Auto-select on single match. Plain text when piped. |
| `rm <name> [--vault]` | Remove by name. |
| `generate [--length n] [--no-symbols]` | Random password, default 20 chars. |
| `export [name] [--vault] [--eval]` | Without `--eval`: output `export KEY='value'` to stdout. With `--eval`: inject env vars. With `--eval -- ./cmd args`: run subprocess with injected env + propagate exit code. No name = all secrets. |
| `import <file> [--vault] [--replace]` | Detect format by extension (.csv/.env). `--replace` overwrites duplicates. Error on unknown extension. |
| `config [[key] value]` | 0 args: list all. 1 arg: show value. 2 args: set value. Keys: `auto-sync` (on/off), `session-timeout` (minutes, default 15), `git` (remote URL). |
| `sync [--vault]` | Git commit + pull --rebase + push. No `--vault` = sync current vault. |
| `passwd [--vault]` | Change password. Empty passwords valid. Min 8 if non-empty. |
| `lock` | Clear all agent cached passwords. |
| `vaults` | List all vault names. |

## Files to DELETE

- `cmd/env.go` — replaced by `export`
- `cmd/exec.go` — replaced by `export --eval`
- `cmd/totp.go` — not in spec
- `cmd/gitremote.go` — replaced by `config git`
- `internal/totp/totp.go` + `totp_test.go` — not in spec

## Files to REWRITE

- `cmd/root.go` — remove `--vault-path`, keep `--vault`
- `cmd/add.go` — prompt only for value (no username/notes/tags/totp)
- `cmd/get.go` — remove username/tags/totp display
- `cmd/list.go` — custom selector (drop promptui), remove tag filter
- `cmd/export.go` — single command with `--eval` + subprocess
- `cmd/import.go` — single command, extension-based format detection, `--replace`
- `cmd/config.go` — add "git" key
- `cmd/sync.go` — remove `--auto` flag
- `cmd/init.go` — allow empty password with warning
- `cmd/passwd.go` — allow empty password
- `cmd/input.go` — simplify password prompts
- `internal/vault/vault.go` — simplified Entry, remove ListByTag/Search complexity
- `internal/vault/storage.go` — bump version to 2
- `internal/sync/config.go` — add "git" key
- `internal/importer/` — env parser stays, CSV becomes simple name,value format

## Files to KEEP (minimal changes)

- `main.go`
- `cmd/generate.go`
- `cmd/lock.go`
- `cmd/vaults.go`
- `cmd/agent.go`
- `internal/crypto/` — unchanged
- `internal/generate/` — unchanged
- `internal/clipboard/` — unchanged
- `internal/agent/` — unchanged
- `internal/sync/git.go` — unchanged

## Dependencies

- Remove: `github.com/manifoldco/promptui`, `github.com/chzyer/readline`
- Keep: `github.com/spf13/cobra`, `github.com/atotto/clipboard`, `golang.org/x/crypto`, `golang.org/x/term`

## Security Considerations

1. Memory zeroing for all passwords/keys/plaintext after use
2. Shell escaping for env var values (prevent injection)
3. File permissions: vault 0600, directories 0700
4. Agent socket: 0600 permissions
5. Atomic writes (temp file + rename) to prevent corruption
6. Advisory flock to prevent concurrent writes
7. Clipboard auto-clear after 30s
8. No secrets in error messages or logs
9. Subprocess env injection: don't expose in /proc/cmdline
10. Import: validate file paths, no path traversal
11. Config values: validate before writing

## Custom Interactive Selector Design

- Use `golang.org/x/term` raw mode
- Parse escape sequences: `\x1b[A`/`\x1bOA` (up), `\x1b[B`/`\x1bOB` (down)
- Handle: Enter (select), Ctrl+C/q (cancel)
- Restore terminal state on signal (SIGINT, SIGTERM)
- Render with ANSI: hide cursor, clear lines, highlight current

## Implementation Order (TDD)

1. `internal/vault/` — Entry model + CRUD
2. `internal/vault/storage.go` — version bump
3. `internal/importer/` — simplified parsers
4. `internal/sync/config.go` — git key
5. `cmd/` helpers — selector, password prompts
6. Commands: init → add → get → list → rm → generate → export → import → config → sync → passwd → lock → vaults
7. Integration test: full workflow
8. Cleanup: remove dead code, go mod tidy

## DRY Helpers Needed

- `loadVault(app) (*vault.Vault, *vault.Store, []byte, error)` — unlock + load pattern
- `saveAndSync(store, vault, password)` — save + auto-sync pattern
- `toEnvVar(name) string` — name → SCREAMING_SNAKE_CASE
- `shellEscape(s) string` — single-quote escaping
- `selector(items []string, out io.Writer, in io.Reader) (int, error)` — interactive picker
