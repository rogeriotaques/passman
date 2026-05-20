# Tech Stack

## Language

- Go 1.25+
- Single binary, no runtime dependencies

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI command framework |
| `github.com/atotto/clipboard` | System clipboard access |
| `golang.org/x/crypto` | Argon2id key derivation |
| `golang.org/x/term` | Terminal raw mode, password reading |

## Crypto

- **Encryption**: AES-256-GCM (authenticated encryption with associated data)
- **Key derivation**: Argon2id (time=3, memory=64 MiB, threads=4, key=32 bytes)
- **Randomness**: `crypto/rand` (CSPRNG) for salts (16 bytes), nonces (12 bytes), passwords
- **Vault format version**: 2 (JSON envelope with KDF params, nonce, ciphertext)

## Architecture

```
main.go              → entrypoint
cmd/                 → CLI commands (cobra), test helpers, input handling
internal/
  agent/             → Session caching daemon (Unix socket)
  clipboard/         → Clipboard copy with auto-clear
  crypto/            → AES-256-GCM + Argon2id
  generate/          → Password generation
  importer/          → .env and CSV parsers
  selector/          → Interactive terminal selector (ANSI + raw mode)
  sync/              → Git operations and config
  vault/             → Vault/Entry types, CRUD, encrypted file I/O
```

## Storage

- Vault files: `~/.passman/vaults/<name>/vault.enc`
- Config files: `~/.passman/vaults/<name>/config.json`
- Agent socket: `~/.passman/agent.sock`
