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
- Agent auto-shuts down when all entries expire

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

- Only `vault.enc` (encrypted) is committed
- Master password never leaves the local machine
- Remote only ever sees encrypted data
