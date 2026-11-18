# Security

## Encryption

- AES-256-GCM (authenticated encryption)
- 32-byte key derived from password via Argon2id
- 12-byte random nonce per encryption (never reused)
- 16-byte random salt per key derivation

## Key Derivation

- Algorithm: Argon2id
- Parameters: time=3, memory=64 MiB, threads=4
- Output: 32-byte key

## File Security

- Vault files: `0600` (owner read/write only)
- Vault directories: `0700` (owner only)
- Atomic writes: write to `.tmp` then `rename()` — prevents corruption on crash
- Advisory flock: prevents concurrent writes from multiple processes

## Agent Security

- Unix domain socket at `~/.passman/agent.sock`
- Socket created with `umask(0077)` — eliminates TOCTOU race on permissions
- Passwords cached in memory with configurable TTL (default 15 minutes)
- Passwords zeroed on expiry, lock, or agent shutdown
- Agent auto-shuts down when all entries expire, unless it can unlock a vault
  on demand for SSH (then ssh must always find its socket)

## Agent Lifecycle

- `Stop` runs once (`sync.Once`), from the reaper, a signal (SIGINT/SIGTERM/
  SIGHUP), or tests; it zeroes secrets and removes both sockets
- `Start` returns only after `Stop` finished: the agent process exits as soon
  as `Start` returns, and returning earlier left a stale `ssh-agent.sock`
  behind (Go's listener `Close` unlinks only its own socket)
- `ssh env` (shell startup) starts the agent when it is down; a failure is a
  warning, never a broken shell

## SSH Agent

- Second Unix socket `~/.passman/ssh-agent.sock`, same umask protection
- Keys are loaded from a vault only after it was unlocked by the user (a
  successful `Store`) or through the on-demand unlock dialog
- Private keys never cross the socket: clients get `List` and `Sign` only;
  `Add`/`Remove`/`Lock` are rejected
- Signatures ask for approval by default (`ssh-confirm on`); the dialog names
  the requesting process (`LOCAL_PEERPID` on macOS, `SO_PEERCRED` on Linux)
  and the key. An approval is remembered for `ssh-confirm-grace` seconds
  (default 60) keyed by the peer's executable path and the key fingerprint —
  never by pid, never for unknown peers; a denial revokes it, lock/unload
  clears it
- On-demand unlock loads keys only — it never caches the master password, so
  triggering the prompt cannot widen a client's access; prompts back off
  exponentially after cancellations (10 s → 15 min) and can be disabled with
  `ssh-unlock-prompt off`
- Dialog text is passed to `osascript` as an argument, never interpolated into
  the script (no AppleScript injection via key or process names)
- `ssh add` reads at most 64 KiB and zeroes the file bytes and passphrase
- `ssh askpass` never replays a vault passphrase for a "Bad passphrase" retry

## Memory Safety

- `vault.ZeroBytes()` zeros password, key, and plaintext byte slices after use
- Deferred zeroing ensures cleanup even on error paths
- Go GC limitation: copies of data may remain in heap (documented trade-off)

## Clipboard

- Auto-clears after 30 seconds
- Only clears if current clipboard content still matches the copied secret

## Error Handling

- Decryption errors are opaque: "wrong password or corrupted data"
- No secrets exposed in error messages or logs
- Invalid passwords don't reveal whether the vault exists or is corrupted

## Input Validation

- Passwords: minimum 8 characters if non-empty, empty is allowed
- Import files: 10 MB size limit
- File extensions: only `.csv` and `.env` accepted for import
- Config values: validated before writing
- Env var names: sanitized to `[A-Z0-9_]` only

## Shell Escaping

- Export values are single-quoted with proper escaping
- Prevents shell injection via secret values
- Format: `export KEY='value'` with `'` escaped as `'"'"'`

## Git Sync

- Only `vault.enc` (encrypted) and `config.json` are committed; `.gitignore`
  excludes `*.lock` and `*.tmp`
- Vault commits are never GPG-signed (`-c commit.gpgsign=false`), so background
  sync cannot hang on a passphrase prompt; `GIT_TERMINAL_PROMPT=0`
- Branch is always `main`; `Sync` pushes first and only pulls (rebase) when
  the push is rejected, so a routine sync costs one SSH connection
- Master password never leaves the local machine
- Remote only ever sees encrypted data
