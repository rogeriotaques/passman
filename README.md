```
 ██████╗  █████╗ ███████╗███████╗███╗   ███╗ █████╗ ███╗   ██╗
 ██╔══██╗██╔══██╗██╔════╝██╔════╝████╗ ████║██╔══██╗████╗  ██║
 ██████╔╝███████║███████╗███████╗██╔████╔██║███████║██╔██╗ ██║
 ██╔═══╝ ██╔══██║╚════██║╚════██║██║╚██╔╝██║██╔══██║██║╚██╗██║
 ██║     ██║  ██║███████║███████║██║ ╚═╝ ██║██║  ██║██║ ╚████║
 ╚═╝     ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝
```

Manage secrets from your terminal.

Passman stores secrets and credentials to be used as environment variables for applications
in an AES-256-GCM encrypted vault with Argon2id key derivation. It supports multiple vaults,
clipboard integration, git sync, and import/export.

## Quick start

```bash
# Create a vault (vault-name is optional, defaults to default)
passman init [vault-name]

# Add a secret
passman add [secret-name] [--vault vault-name]

# Retrieve a secret
passman get [secret-name] [--vault vault-name]

# List secret names
passman list [secret-name] [--vault vault-name]

# List all vaults
passman vaults

# Remove a secret by its name
passman rm [secret-name] [--vault vault-name]

# Generate a random password
passman generate [--length <n>] [--no-symbols]

# Export secrets as environment variables or inject with --eval
passman export [secret-name] [--vault vault-name] [--eval]

# Import a CSV or .env file
passman import file.csv|.env [--vault vault-name] [--replace]

# View or update vault configuration
passman config [[key] value]

# Sync/backup the vault with its remote
passman sync [--vault vault-name]

# Change the vault password
passman passwd [--vault vault-name]

# Clear cached passwords from the agent
passman lock

# Permanently delete a vault (or all vaults)
passman destroy [--vault vault-name]
```

Vaults are stored as `~/.passman/vaults/<name>/vault.enc`.

## Install

```bash
go build -o passman .
```

Move the binary somewhere in your `$PATH`:

```bash
mv passman /usr/local/bin/
```

## Features

- AES-256-GCM encrypted vault with Argon2id key derivation
- Clipboard integration with 30-second auto-clear
- Cryptographically secure password generation
- Git-based vault backup and sync across devices
- Background auto-sync on vault changes
- Shell integration — inject secrets as environment variables
- Import from .env files and CSV
- Export to .env format
- Multiple named vaults
- Case-insensitive entry lookup
- Session caching via background agent
- Single binary, no external dependencies at runtime

## Security

- **Encryption**: AES-256-GCM (authenticated encryption)
- **Key derivation**: Argon2id (time=3, memory=64 MiB, threads=4)
- **Random generation**: `crypto/rand` (CSPRNG) for all randomness — salts, nonces, passwords
- **File permissions**: Vault file created with `0600`, directories with `0700`
- **Clipboard**: Auto-clears after 30 seconds
- **Agent**: Unix socket with `0600` permissions, umask-protected creation
- **Atomic writes**: vault written to temp file then renamed
- **File locking**: advisory flock prevents concurrent writes
- **Memory zeroing**: passwords, keys, and plaintext zeroed after use
- **Error opacity**: Decryption errors don't distinguish wrong password from corruption

## Development

Run all tests:

```bash
go test ./...
```

See [docs/DOCS.md](docs/DOCS.md) for full command reference.

## License

MIT
