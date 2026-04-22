```
 ██████╗  █████╗ ███████╗███████╗███╗   ███╗ █████╗ ███╗   ██╗
 ██╔══██╗██╔══██╗██╔════╝██╔════╝████╗ ████║██╔══██╗████╗  ██║
 ██████╔╝███████║███████╗███████╗██╔████╔██║███████║██╔██╗ ██║
 ██╔═══╝ ██╔══██║╚════██║╚════██║██║╚██╔╝██║██╔══██║██║╚██╗██║
 ██║     ██║  ██║███████║███████║██║ ╚═╝ ██║██║  ██║██║ ╚████║
 ╚═╝     ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝
```

Manage passwords, secrets, and 2FA from your terminal. Store, retrieve, and generate credentials from an encrypted local vault with Git-based sync, shell integration, TOTP, and multi-vault support.

## Why Passman

Most password managers are built for browsers. Passman is built for terminals.

If you deploy from the command line, rotate keys in scripts, or SSH into machines where a GUI doesn't exist, you've probably resorted to `.env` files scattered across directories, secrets hard-coded in shell history, or copy-pasting from a browser extension into a terminal tab. None of that is great.

Passman gives you an encrypted vault that fits into the workflows you already have:

- **Pipe secrets into commands** without exposing them in shell history or process lists: `passman exec prod -- terraform apply`
- **Source credentials as environment variables** in any shell session: `eval "$(passman env staging)"`
- **Automate credential rotation** in scripts — generate, store, and retrieve without leaving the terminal
- **Sync your vault across machines** via Git, with nothing but the encrypted file ever touching the remote
- **Import your existing credentials** from 1Password, Bitwarden, or `.env` files and keep working from the CLI

It's a single binary with no runtime dependencies, no cloud account, and no browser required. Your secrets stay local, encrypted at rest, and accessible wherever you have a shell.

## Features

- AES-256-GCM encrypted vault with Argon2id key derivation
- Clipboard integration with 30-second auto-clear
- Cryptographically secure password generation
- Git-based vault backup and sync across devices
- Background auto-sync on vault changes
- Shell integration — inject secrets as environment variables
- TOTP code generation (RFC 6238)
- Import from .env files and 1Password/Bitwarden CSV exports
- Export to .env and CSV formats
- Multiple named vaults with automatic migration
- Case-insensitive entry lookup
- Single binary, no external dependencies at runtime

## Install

```bash
go build -o passman .
```

Move the binary somewhere in your `$PATH`:

```bash
mv passman /usr/local/bin/
```

## Usage

### Initialize a vault

```bash
passman init
passman init --vault prod
passman init --git
```

Creates an encrypted vault. You'll be prompted for a master password. Use `--vault` to create a named vault (default: "default"). Use `--git` to also set up Git sync with an optional remote URL prompt.

### Add a credential

```bash
passman add github
passman add aws-key --tag prod --tag infra
passman add github --totp
```

Prompts for master password, then username, password, and optional notes. Use `--tag` to categorize entries (repeatable). Use `--totp` to also store a TOTP secret for 2FA code generation.

### Retrieve a credential

```bash
passman get github
passman get --print github
```

Copies the password to your clipboard and clears it after 30 seconds. Use `--print` to output to stdout instead. Displays tags and TOTP status if configured.

### List entries

```bash
passman list
passman list --vault prod
```

Shows entry names alphabetically. Never displays secrets.

### Remove an entry

```bash
passman rm github
```

### Generate a password

```bash
passman generate
passman generate --length 32
passman generate --no-symbols
```

Generates a cryptographically random password. Default: 20 characters with all character sets.

### TOTP codes

```bash
passman totp github
# 483291 (expires in 18s)
```

Generates a time-based one-time password for entries with a TOTP secret configured.

### Shell integration

Export vault entries as environment variables:

```bash
eval $(passman env)              # all entries
eval $(passman env prod)         # entries tagged "prod"
passman env staging > .env       # write .env file
```

Entry names are converted to env var format: uppercased, hyphens/spaces become underscores (e.g. `aws-key` becomes `AWS_KEY`).

Run a command with secrets injected (never touches shell history):

```bash
passman exec prod -- ./deploy.sh
passman exec -- docker-compose up
```

### Import / Export

Import from external sources:

```bash
passman import env .env.production
passman import csv 1password-export.csv
```

CSV import supports 1Password and Bitwarden export formats. Duplicate entries are skipped.

Export your vault:

```bash
passman export env > .env
passman export env prod > .env.production
passman export csv > backup.csv
```

### Sync vault

```bash
passman sync
```

Commits local changes, pulls from remote (with rebase), and pushes.

#### Auto-sync

```bash
passman sync --auto on
passman sync --auto off
```

When enabled, sync runs in the background after every `add`, `rm`, or `import`. Failures produce a warning but never block the vault operation.

### Manage Git remote

```bash
passman git remote <url>
```

### Multiple vaults

```bash
passman init --vault prod
passman add aws-key --vault prod
passman list --vault staging
passman env --vault prod
passman vaults                   # list all vault names
```

Each vault has its own master password, git remote, and auto-sync config. Vaults are stored under `~/.passman/vaults/<name>/`. Existing single-vault installations are automatically migrated to the "default" vault on first run.

## Options

| Flag | Scope | Description |
|---|---|---|
| `--vault` | Global | Vault name (default: "default") |
| `--vault-path` | Global | Direct path to vault file (overrides `--vault`) |
| `--git` | `init` | Initialize Git repository for vault sync |
| `--tag` | `add` | Tag for the entry (repeatable) |
| `--totp` | `add` | Prompt for TOTP secret |
| `--length` | `generate` | Password length (default: 20) |
| `--no-symbols` | `generate` | Exclude symbols from generated password |
| `-p, --print` | `get` | Print password to stdout instead of clipboard |
| `--auto` | `sync` | Toggle auto-sync (`on` or `off`) |

## Security

- **Encryption**: AES-256-GCM (authenticated encryption)
- **Key derivation**: Argon2id (time=3, memory=64 MiB, threads=4)
- **Random generation**: `crypto/rand` (CSPRNG) for all randomness — salts, nonces, passwords
- **File permissions**: Vault file created with `0600`
- **Clipboard**: Auto-clears after 30 seconds, only if clipboard content hasn't changed
- **Error opacity**: Decryption errors don't distinguish wrong password from corruption
- **Safe to sync**: Vault file is always encrypted on disk
- **TOTP caveat**: Storing passwords and TOTP secrets together means a vault compromise exposes both factors

### Limitations

- Go's garbage collector may retain copies of secrets in memory
- Clipboard managers may capture secrets before auto-clear
- No file locking — single-user, single-process assumed
- Git sync uses rebase strategy — concurrent edits on multiple devices may require manual conflict resolution

## Development

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test ./... -v
```

### Project structure

```
passman/
  main.go                       # entrypoint
  cmd/                          # CLI commands (cobra)
    root.go                     # root command, App struct, vault resolution, migration
    init.go                     # passman init (--git, --vault)
    add.go                      # passman add (--tag, --totp, auto-sync)
    get.go                      # passman get (clipboard, tags, TOTP indicator)
    list.go                     # passman list
    rm.go                       # passman rm (auto-sync)
    generate.go                 # passman generate
    env.go                      # passman env (shell integration)
    exec.go                     # passman exec (subprocess with injected env)
    totp.go                     # passman totp (2FA code generation)
    import.go                   # passman import (env, csv)
    export.go                   # passman export (env, csv)
    sync.go                     # passman sync (manual + --auto toggle)
    gitremote.go                # passman git remote
    vaults.go                   # passman vaults (list all vaults)
    input.go                    # password/input reading helpers
    cmd_test.go                 # CLI integration tests
  internal/
    crypto/                     # AES-256-GCM encryption, Argon2id KDF
    vault/                      # Vault/Entry types, CRUD, encrypted file I/O
    generate/                   # Password generation
    clipboard/                  # Clipboard copy with auto-clear
    sync/                       # Git operations and config for vault sync
    totp/                       # TOTP code generation (RFC 6238)
    importer/                   # .env and CSV file parsers
```

## License

MIT
