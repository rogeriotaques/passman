# Passman

Manage passwords, secrets, and 2FA from your terminal.

Passman stores credentials in an AES-256-GCM encrypted vault with Argon2id key
derivation. It supports multiple vaults, TOTP, clipboard integration, password
caching, git sync, and import/export.

## Quick start

```bash
# Create a vault
passman init

# Add a credential
passman add github

# Retrieve it (copies password to clipboard, clears in 30s)
passman get github

# List entries
passman list
```

## Global flags

| Flag | Description |
|---|---|
| `--vault <name>` | Use a named vault (default: `default`) |
| `--vault-path <path>` | Use a custom vault file path |

Vaults are stored under `~/.passman/vaults/<name>/vault.enc`.

## Commands

### init

Create a new encrypted vault. You will be prompted for a master password
(minimum 8 characters) with confirmation.

```bash
passman init
passman init --vault work
passman init --git
```

| Flag | Description |
|---|---|
| `--git` | Initialize a git repository for vault sync |

With `--git`, you are optionally prompted for a remote URL.

---

### add

Add a new credential to the vault. You will be prompted for username, password,
and optional notes.

```bash
passman add github
passman add aws-prod --tag aws --tag production
passman add gitlab --totp
```

| Flag | Description |
|---|---|
| `--tag <value>` | Tag for the entry (can be repeated) |
| `--totp` | Prompt for a TOTP secret (base32) |

---

### get

Retrieve a credential by name. The password is copied to the clipboard and
automatically cleared after 30 seconds.

```bash
passman get github
passman get github --print
```

| Flag | Description |
|---|---|
| `-p`, `--print` | Print password to stdout instead of clipboard |

---

### list

List and search vault entries. Without arguments, shows the first 20 entries.
With arguments, performs a multi-token search across entry names, usernames,
notes, and tags. All tokens must match (case-insensitive) for an entry to
appear.

In a terminal, results are shown as an interactive selector — use arrow keys
to navigate and Enter to retrieve the selected entry (password is copied to
clipboard). If only one entry matches, it is auto-selected. When piped,
output is plain text (one name per line).

```bash
passman list
passman list --all
passman list github
passman list alice@ github
passman list --tag prod
```

| Flag | Default | Description |
|---|---|---|
| `-a`, `--all` | false | Show all entries without truncation |
| `--tag <value>` | | Filter by tag |

---

### rm

Permanently remove a credential from the vault by name.

```bash
passman rm old-service
```

---

### generate

Generate a cryptographically random password. The password is guaranteed to
contain at least one character from each enabled class (uppercase, lowercase,
digits, symbols).

```bash
passman generate
passman generate --length 32
passman generate --no-symbols
```

| Flag | Default | Description |
|---|---|---|
| `--length <n>` | 20 | Password length |
| `--no-symbols` | false | Exclude symbols |

---

### totp

Generate a time-based one-time password (TOTP) for an entry that has a TOTP
secret configured (see `passman add --totp`).

```bash
passman totp github
```

Output: `123456 (expires in 18s)`

---

### env

Print vault entries as shell `export` statements. Entry names are converted to
`SCREAMING_SNAKE_CASE`. Optionally filter by tag.

```bash
passman env
passman env aws
eval "$(passman env production)"
```

Output format: `export MY_SECRET='value'`

---

### exec

Run a command with vault secrets injected as environment variables. All current
environment variables are preserved; vault entries are added on top.

Use a tag before `--` to inject only matching entries. Everything after `--` is
the command and its arguments.

```bash
passman exec -- ./deploy.sh
passman exec aws -- terraform apply
passman exec -- env | grep AWS
```

---

### export

Export vault credentials to external formats.

#### export env

Export as `.env` format (`KEY='value'` pairs, no `export` prefix). Optionally
filter by tag.

```bash
passman export env > .env
passman export env aws > .env.aws
```

#### export csv

Export all entries as CSV with columns: Name, Username, Password, Notes, Tags.

```bash
passman export csv > backup.csv
```

---

### import

Import credentials into the vault. Duplicate names are skipped.

#### import env

Import from a `.env` file. Each `KEY=VALUE` line becomes a vault entry.

```bash
passman import env .env
passman import env /path/to/secrets.env
```

#### import csv

Import from a 1Password or Bitwarden CSV export.

```bash
passman import csv ~/Downloads/1password-export.csv
passman import csv bitwarden.csv
```

---

### config

View or update vault configuration.

```bash
passman config                        # show all settings
passman config session-timeout        # show one setting
passman config session-timeout 30     # set a value
passman config auto-sync on           # enable auto-sync
```

| Key | Values | Default | Description |
|---|---|---|---|
| `auto-sync` | `on` / `off` | `off` | Automatically sync after writes |
| `session-timeout` | minutes | `15` | Minutes before cached password expires (0 to disable) |

---

### sync

Sync the vault with its git remote (commit + pull + push). The vault must have
been initialized with `--git`.

```bash
passman sync
passman sync --auto on
passman sync --auto off
```

| Flag | Description |
|---|---|
| `--auto <on\|off>` | Toggle automatic sync after every write |

---

### git remote

Set or change the git remote URL for the vault repository.

```bash
passman git remote git@github.com:user/vault.git
```

---

### passwd

Change the vault master password. You will be prompted for the current
password, then a new password (minimum 8 characters) with confirmation. The
vault is decrypted with the old password and re-encrypted with the new one.
Any cached password in the agent is cleared.

```bash
passman passwd
```

---

### lock

Immediately clear all cached master passwords from the background agent. The
next vault operation will prompt for the password again.

```bash
passman lock
```

---

### vaults

List all vaults found under `~/.passman/vaults/`.

```bash
passman vaults
```

## Session caching

Passman caches the master password in a background agent (similar to
`ssh-agent`) so you don't have to re-enter it for every command.

- The agent communicates over a Unix domain socket (`~/.passman/agent.sock`)
- The socket is created with `0600` permissions (owner-only access)
- Passwords are held in memory with a configurable TTL
- The agent auto-starts on first use and auto-shuts down when all entries expire

### Configure the timeout

```bash
# Set to 30 minutes
passman config session-timeout 30

# Disable caching (always prompt)
passman config session-timeout 0
```

Default: 15 minutes.

### Lock immediately

```bash
passman lock
```

## Git sync

Passman can sync the encrypted vault via git, allowing you to share it across
machines.

```bash
# Initialize with git support
passman init --git

# Set a remote (if not done during init)
passman git remote git@github.com:user/vault.git

# Manual sync
passman sync

# Enable auto-sync after every write (add, rm, import)
passman sync --auto on
```

Only the encrypted `vault.enc` file is committed. Your master password never
leaves the local machine.

## Multiple vaults

```bash
# Create a work vault
passman init --vault work

# Use it
passman add github --vault work
passman get github --vault work
passman list --vault work

# List all vaults
passman vaults
```

## Security

- **Encryption**: AES-256-GCM with Argon2id key derivation
- **Atomic writes**: vault is written to a temp file then renamed, preventing
  corruption
- **File locking**: advisory flock prevents concurrent writes
- **Memory zeroing**: passwords, keys, and plaintext are zeroed after use
- **Clipboard clearing**: passwords auto-clear from clipboard after 30 seconds
- **Agent security**: Unix socket with 0600 permissions, passwords zeroed on
  expiry and lock
