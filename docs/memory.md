# Work Log

## 2026-05-20: Major Refactoring (POC → DOCS.md Spec)

### Completed

1. **Simplified Entry model** — `{Name, Username, Password, Notes, Tags, TotpSecret, CreatedAt, UpdatedAt}` → `{Name, Value}`
2. **Bumped vault version** — 1 → 2 (clean break, no migration)
3. **Rewrote importers** — .env parser uses Value field, CSV parser expects name,value columns
4. **Added "git" config key** — remote URL stored in config.json
5. **Fixed agent socket security** — umask(0077) before Listen eliminates TOCTOU race
6. **Built custom interactive selector** — replaced promptui with raw terminal + ANSI
7. **Rewrote all commands** to match DOCS.md spec:
   - init: accepts optional vault-name positional arg, empty password allowed
   - add: prompts only for secret value
   - get: simplified output (just value)
   - list: name-only search, custom selector, --all flag
   - export: single command with --eval, subprocess execution, exit code propagation
   - import: extension-based format detection (.csv/.env), --replace flag, 10MB limit
   - config: added "git" key, sets git remote when in a repo
   - sync: removed --auto flag (controlled via config)
   - passwd: allows empty passwords
8. **Removed dead code**:
   - cmd/env.go, cmd/exec.go, cmd/totp.go, cmd/gitremote.go
   - internal/totp/ package
   - storePasswordInAgent() wrapper
   - fileExists() in sync/config.go
   - ListByTag(), Tags, Username, Notes, TotpSecret fields
9. **Removed dependencies** — promptui, chzyer/readline
10. **Added DRY helpers** — loadVault(), saveAndSync(), copyToClipboard()
11. **Created documentation** — AGENTS.md, CLAUDE.md, docs/knowledge/*, docs/memory.md

### Post-refactor fixes

- **list**: removed auto-copy on single match — user must press ENTER in selector
- **empty password**: added `no_password` config flag so subsequent commands skip prompt
- **error output**: silenced cobra usage dump, errors print as sentence-case to stderr
- **completion**: disabled cobra's built-in `completion` command (not in spec)
- **export**: removed `export` prefix from default output (for `.env` file compat); `--eval` keeps it for shell eval; fixed `DisableFlagParsing` so `-- command` args work
- **destroy**: added `passman destroy [--vault name]` to permanently delete vaults with confirmation

### Test Results

All packages pass: cmd, agent, clipboard, crypto, generate, importer, selector, sync, vault
