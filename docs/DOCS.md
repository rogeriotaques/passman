# Passman

Manage secrets from your terminal.

Passman stores secrets and credentials to be used as environment variables for applications
in an AES-256-GCM encrypted vault with Argon2id key derivation. It supports multiple vaults,
clipboard integration, git sync, and import/export.

## Quick start

Commands are grouped by noun: `secret` for entries, `vault` for the vault itself.

```bash
# Create a vault (vault-name is optional, defaults to default)
passman vault init [vault-name]

# Add a secret
passman secret add [secret-name] [--vault vault-name]

# Retrieve a secret
passman secret get [secret-name] [--vault vault-name]

# List secret names
passman secret list [query...] [--vault vault-name]

# Remove a secret by its name
passman secret remove [secret-name] [--vault vault-name]

# Export secrets or inject them into a command
passman secret export [secret-name] [--vault vault-name] [--eval]

# Import a CSV or .env file
passman secret import file.csv|.env [--vault vault-name] [--replace]

# List all vaults
passman vault list

# View or update vault configuration
passman vault config [[key] value]

# Sync the vault with its remote
passman vault sync [--vault vault-name]

# Change the vault password
passman vault passwd [--vault vault-name]

# Permanently delete a vault (or all vaults)
passman vault destroy [--vault vault-name]

# Generate a random password (no vault needed)
passman generate [--length <n>] [--no-symbols]

# Immediately lock the background agent and clear cached passwords
passman lock

# Upgrade passman to the latest version
passman upgrade
```

Vaults are stored as `~/.passman/vaults/<name>/vault.enc`.

## Command overview

| Group | Commands |
| ----- | -------- |
| `passman secret` | `add`, `get`, `list`, `remove`, `export`, `import` |
| `passman vault` | `init`, `list`, `passwd`, `config`, `sync`, `destroy` |
| `passman` | `generate`, `lock`, `upgrade` |

## Commands

### secret add

Add a new secret to the vault. You will be prompted for the value.

```bash
passman secret add github
passman secret add github --vault work
```

---

### secret get

Retrieve a secret by its name. The value is copied to the clipboard and
automatically cleared after 30 seconds.

```bash
passman secret get github
passman secret get github --vault work

# Print the value instead of copying it
passman secret get github --print
```

---

### secret list

List and query vault entries.

Without arguments, shows the first 20 entries. With arguments, performs a
multi-token search across entry names. All tokens must match (case-insensitive)
for an entry to appear.

When searching, results are shown as an interactive selector — use arrow keys
to navigate and `ENTER` to retrieve the selected secret. If only one entry matches,
it is auto-selected. When piped, output is plain text (one name per line).

```bash
# List 20 first entries with interactive selector
passman secret list

# List all entries with interactive selector
passman secret list --all

# List entries matching a search query with interactive selector
passman secret list github
passman secret list alice@ github

# List 20 first entries from the work vault with interactive selector
passman secret list --vault work
```

---

### secret remove

Permanently remove a secret from the vault by name.

```bash
# Remove the entry from the default vault
passman secret remove secret-name

# Remove the entry from the work vault
passman secret remove secret-name --vault work
```

---

### secret export

Export entries as shell variables, or inject them into a command with `--eval`.

```bash
passman secret export github > .env
passman secret export github --vault work > .env
passman secret export github [--vault work] --eval -- ./deploy.sh
```

Entry names are converted to `SCREAMING_SNAKE_CASE`. When `--eval` flag is passed,
secrets are injected as environment variables. Existing environment variables are
preserved, and new ones are injected on top.

---

### secret import

Import secrets into the vault.

```bash
# Import an .env file into the default vault
passman secret import .env

# Import a CSV file into the work vault
passman secret import file.csv --vault work

# Import and replace duplicate values
passman secret import file.csv --vault work --replace
```

---

### vault init

Create a new encrypted vault. You will be prompted for a master password
(minimum 8 characters) with confirmation. If not provided, the vault is
created without password.

```bash
# Creates the default vault
passman vault init

# Creates a vault called "work"
passman vault init work

# Creates the default vault and initializes the git repository to sync/backup
passman vault init --git
```

| Flag | Description |
| ---- | ----------- |
| `--git` | Initialize a git repository for vault sync |

With `--git`, you are optionally prompted for a remote URL. All git-based
repositories are supported.

---

### vault list

List all vaults found under `~/.passman/vaults/`.

```bash
passman vault list
```

---

### vault config

View or update vault configuration.

```bash
# List all configuration values
passman vault config

# Show one setting
passman vault config session-timeout

# Set a value
passman vault config session-timeout 30
```

| Key | Values | Default | Description |
| --- | ------ | ------- | ----------- |
| `auto-sync` | `on` / `off` | `off` | Automatically sync after writes |
| `session-timeout` | minutes | `15` | Minutes before cached password expires (0 to disable) |
| `git` | set `git@github.com:user/vault.git` | | List or set the git remote URL for the vault repository |

---

### vault sync

Passman can sync the encrypted vault via git, allowing you to share it across
machines. Sync the vault with its git remote (commit + pull + push). The vault must have
been initialized with `--git`.

```bash
# Sync the default vault
passman vault sync

# Sync the work vault
passman vault sync --vault work
```

```bash
# Initialize with git support
passman vault init --git

# Set a remote (if not done during init)
passman vault config git git@github.com:user/vault.git

# Manual sync
passman vault sync

# Enable auto-sync after every write (add, remove, import)
passman vault config auto-sync on
```

Only the encrypted `vault.enc` file is committed. Your master password never
leaves the local machine.

---

### vault passwd

Change the vault master password. You will be prompted for the current
password, then a new password (minimum 8 characters) with confirmation. The
vault is decrypted with the old password and re-encrypted with the new one.
Any cached password in the agent is cleared. Empty passwords are valid.

```bash
passman vault passwd
```

---

### vault destroy

Permanently delete a vault or all vaults. You will be prompted to type
`yes` to confirm the operation.

```bash
# Destroy a specific vault
passman vault destroy --vault work

# Destroy all vaults
passman vault destroy
```

Without `--vault`, all vaults are deleted. This operation is irreversible.

---

### generate

Generate a cryptographically random password. The password is guaranteed to
contain at least one character from each enabled class (uppercase, lowercase,
digits, symbols).

```bash
# Generate a random password
passman generate

# Generate a random password with custom length
passman generate --length 32

# Generate a random password with no symbols
passman generate --no-symbols
```

---

### lock

Immediately clear all cached master passwords from the background agent. The
next vault operation will prompt for the password again.

```bash
passman lock
```

---

### upgrade

Upgrade passman to the latest version by cloning the repository, compiling, and replacing the current binary.

```bash
passman upgrade
```

Requires `git` and `go` to be installed and available in your PATH.

The version is read from the `VERSION` file (semver) on the `master` branch. To ship a new version, update `VERSION` before committing: the value is embedded in the binary at build time, and `upgrade` uses it to detect whether an update is available.

The value must be strict semver (`MAJOR.MINOR.PATCH`, with optional pre-release/build). An invalid or empty `VERSION` makes the binary exit with an error, and `upgrade` refuses to build a clone with a bad `VERSION`.

---

## Session caching

Passman caches the master password in a background agent (similar to
`ssh-agent`) so you don't have to re-enter it for every command.

- The agent communicates over a Unix domain socket (`~/.passman/agent.sock`)
- The socket is created with `0600` permissions (owner-only access)
- Passwords are held in memory with a configurable TTL
- The agent auto-starts on first use and auto-shuts down when all entries expire

## Security

- **Encryption**: AES-256-GCM with Argon2id key derivation
- **Atomic writes**: vault is written to a temp file then renamed, preventing
  corruption
- **File locking**: advisory flock prevents concurrent writes
- **Memory zeroing**: passwords, keys, and plaintext are zeroed after use
- **Clipboard clearing**: passwords auto-clear from clipboard after 30 seconds
- **Agent security**: Unix socket with 0600 permissions, passwords zeroed on
  expiry and lock
