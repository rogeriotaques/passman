# Passman

Manage secrets from your terminal.

Passman stores secrets and credentials to be used as environment variables for applications 
in an AES-256-GCM encrypted vault with Argon2id key derivation. It supports multiple vaults, 
clipboard integration, git sync, and import/export.

## Quick start

```bash
# Create a vault (vault-name is optional, defaults to default)
passman init [vault-name]

# Add a secret
passman add [secret-name]  [--vault vault-name]

# Retrieve a secret
passman get [secret-name]  [--vault vault-name]

# List secret names
passman list [secret-name] [--vault vault-name]

# List all vaults
passman vaults

# Remove a secret by its name
passman rm [secret-name] [--vault vault-name]

# Generate a random password
passman generate [--length <n>] [--no-symbols]

# Export secrets to external formats or inject secrets as environment variables with --eval flag
passman export [secret-name] [--vault vault-name] [--eval]

# Import a CSV or .ENV file
passman import file.csv|.env [--vault vault-name] [--replace]

# View or update vault configuration
passman config [[key] value]

# Sync/backup the vault with its remote.
passman sync [--vault vault-name]

# Change the vault password
passman passwd [--vault vault-name]

# Immediately locks the background agent and clears cached password
passman lock

# Permanently delete a vault (or all vaults)
passman destroy [--vault vault-name]
```

Vaults are stored as `~/.passman/vaults/<name>/vault.enc`.

## Commands

### init

Create a new encrypted vault. You will be prompted for a master password
(minimum 8 characters) with confirmation. If not provided, the vault is 
created without password.

```bash
# Creates the default vault
passman init

# Creates a vault called "work"
passman init work

# Creates the default vault and initialized the git repository to sync/backup
passman init --git
```

| Flag | Description |
| ---- | ----------- |
| `--git` | Initialize a git repository for vault sync |

With `--git`, you are optionally prompted for a remote URL. All git-based 
repositories are supported.

---

### add

Add a new secret to the vault. You will be prompted for the value.

```bash
passman add github
passman add github --vault work
```

---

### get

Retrieve a secret by its name. The value is copied to the clipboard and
automatically cleared after 30 seconds.

```bash
passman get github
passman get github --vault work

# Print a copied value
passman get github --print
```

---

### list

List and query vault entries. 

Without arguments, shows the first 20 entries. With arguments, performs a 
multi-token search across entry names. All tokens must match (case-insensitive)
for an entry to appear.

When searching, results are shown as an interactive selector — use arrow keys
to navigate and `ENTER` to retrieve the selected secret. If only one entry matches,
it is auto-selected. When piped, output is plain text (one name per line).

```bash
# List 20 first entries with interactive selector
passman list

# List all entries with interactive selector
passman list --all

# List entries matching a search query with interactive selector
passman list github
passman list alice@ github

# List 20 first entries from the work vault with interactive selector
passman list --vault work
```

---

### rm

Permanently remove a secret from the vault by name.

```bash
# Remove the entry from the default vault
passman rm secret-name

# Remove the entry from the work vault
passman rm secret-name --vault work
```

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

### export

```bash
passman export github > .env
passman export github --vault work > .env
passman export github [--vault work] --eval -- ./deploy.sh
```

Entry names are converted to `SCREAMING_SNAKE_CASE`. When `--eval` flag is passed, 
secrets are injected as environment variables. Existing environment variables are 
preserved, and new ones are injected on top.

---

### import

Import secrets into the vault.

```bash
# Import an .env file into the default vault
passman import .env

# Import a CSV file into the work vault
passman import file.csv --vault work

# Import and replace duplicate values
passman import file.csv --vault work --replace
```

---

### config

View or update vault configuration.

```bash
# List all configuration values
passman config

# Show one setting
passman config session-timeout

# Set a value
passman config session-timeout 30
```

| Key | Values | Default | Description |
| --- | ------ | ------- | ----------- |
| `auto-sync` | `on` / `off` | `off` | Automatically sync after writes |
| `session-timeout` | minutes | `15` | Minutes before cached password expires (0 to disable) |
| `git` | set `git@github.com:user/vault.git` | | List or set the git remote URL for the vault repository |

---

### sync


Passman can sync the encrypted vault via git, allowing you to share it across
machines. Sync the vault with its git remote (commit + pull + push). The vault must have
been initialized with `--git`.

```bash
# Sync all vaults
passman sync

# Sync a specific vault
passman sync work
```

```bash
# Initialize with git support
passman init --git

# Set a remote (if not done during init)
passman config git git@github.com:user/vault.git

# Manual sync
passman sync

# Enable auto-sync after every write (add, rm, import)
passman config auto-sync on
```

Only the encrypted `vault.enc` file is committed. Your master password never
leaves the local machine.

---

### passwd

Change the vault master password. You will be prompted for the current
password, then a new password (minimum 8 characters) with confirmation. The
vault is decrypted with the old password and re-encrypted with the new one.
Any cached password in the agent is cleared. Empty passwords are valid.

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

---

### destroy

Permanently delete a vault or all vaults. You will be prompted to type
`yes` to confirm the operation.

```bash
# Destroy a specific vault
passman destroy --vault work

# Destroy all vaults
passman destroy
```

Without `--vault`, all vaults are deleted. This operation is irreversible.

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
