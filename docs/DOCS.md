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

# Use the vault as an SSH agent: import keys, list them, wire ssh to passman
passman ssh add <private-key-file> [--name NAME] [--vault vault-name]
passman ssh list [--vault vault-name]
eval "$(passman ssh env)"

# Immediately lock the background agent and clear cached passwords and SSH keys
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
| `passman ssh` | `add`, `list`, `env`, `askpass` |
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
| `git` | set `git@github.com:user/vault.git` | | Set the git remote for the vault (initialises the repository on branch `main` if needed) |
| `ssh-timeout` | minutes | `480` | Minutes the agent serves this vault's SSH keys after an unlock (0 disables SSH keys for the vault) |
| `ssh-confirm` | `on` / `off` | `on` | Ask for approval (dialog) before SSH signatures |
| `ssh-confirm-grace` | seconds | `60` | After an approval, the same program may use the same key without asking for this long (0 asks every time) |
| `ssh-unlock-prompt` | `on` / `off` | `on` | Let the agent offer to unlock this vault (dialog) when ssh finds no keys loaded |

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

### ssh add

Import an SSH private key file into the vault so the agent can serve it. A
passphrase-protected file is decrypted (you are asked for its passphrase) and
stored without it: the vault's encryption protects it from here on. The file
itself is left untouched; delete it once you have a backup.

```bash
# Stored as ssh/id_ed25519
passman ssh add ~/.ssh/id_ed25519

# Custom entry name
passman ssh add ~/.ssh/aws.pem --name ssh/aws-prod
```

The command prints the key's fingerprint and public key line. Supported key
types: ed25519, RSA, ECDSA (anything `ssh.MarshalPrivateKey` understands).
Files larger than 64 KiB are rejected.

---

### ssh list

Show the name, type, and fingerprint of every private key in the vault.
Unlocking the vault for this command also (re)loads the keys into the running
agent.

```bash
passman ssh list
```

---

### ssh env

Print the exports that route ssh through passman and (re)write the small
`SSH_ASKPASS` helper at `~/.passman/askpass`:

```bash
eval "$(passman ssh env)"     # add this line to ~/.zshrc or ~/.bashrc
```

Output:

```bash
export SSH_AUTH_SOCK='/Users/you/.passman/ssh-agent.sock'
export SSH_ASKPASS='/Users/you/.passman/askpass'
export SSH_ASKPASS_REQUIRE='force'
```

`force` is required because ssh only consults `SSH_ASKPASS` when `DISPLAY` is
set otherwise, which terminals on macOS never do.

`ssh env` also starts the agent when it is not running (after a reboot or
logout), so `SSH_AUTH_SOCK` always points at a live socket. The agent then
offers to unlock the default vault the first time a client lists keys (ssh
connecting, `ssh-add -l`). With `force`, every ssh
prompt (passphrases, host key confirmations, passwords) goes through
`passman ssh askpass`, which answers on your terminal when there is one and
through a dialog when there is not.

---

### ssh askpass

The `SSH_ASKPASS` implementation; ssh runs it, never you. It receives the prompt
as its only argument and prints the answer:

- `Enter passphrase for key '<file>'` prompts are answered from the vault entry
  `ssh/passphrase/<file name>` when it exists (for example
  `ssh/passphrase/id_ed25519`). Unlocking the vault may ask for the master
  password first. Retry prompts (`Bad passphrase, try again`) always ask you,
  so a stale vault value cannot loop.
- Other secret prompts ask you (terminal or dialog).
- `SSH_ASKPASS_PROMPT=none` prompts (host key confirmations) show a visible
  input.
- `SSH_ASKPASS_PROMPT=confirm` prompts show a Deny/Allow dialog and exit `0`
  for Allow, `1` for Deny.

---

### lock

Immediately clear all cached master passwords and loaded SSH keys from the
background agent. The next vault operation will prompt for the password again,
and ssh finds no keys until a vault is unlocked.

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
  (unless it serves SSH, see below)

## SSH agent

The same background agent serves the SSH agent protocol on
`~/.passman/ssh-agent.sock`. Point `SSH_AUTH_SOCK` at it with
`eval "$(passman ssh env)"`.

- Whenever a vault is unlocked (any command that reads it), the agent loads the
  vault's private keys and serves them for `ssh-timeout` minutes (default 8
  hours). `passman lock` drops them immediately.
- Private keys never leave the agent. ssh only receives signatures, and with
  `ssh-confirm on` (the default) a signature pops a Deny/Allow dialog naming
  the requesting process and the key. One approval covers the same program
  (same executable path, e.g. `/usr/bin/ssh`) and key for `ssh-confirm-grace`
  seconds, so a `git pull && git push` asks once, not three times. A denial
  revokes the approval; `passman lock` clears them all.
- If ssh asks for keys while none are loaded, the agent offers to unlock the
  default vault through a dialog. That unlock loads SSH keys only: the master
  password is not cached for other commands, so a program connecting to the
  socket cannot widen its access by triggering the prompt. Prompts are
  rate-limited: 10 seconds, doubling after each cancelled prompt (up to 15
  minutes), reset by a successful unlock. Turn the prompt off with
  `passman vault config ssh-unlock-prompt off` and unlock explicitly with
  `passman ssh list` instead.
- `ssh-add` cannot add, remove, or lock keys in this agent: the vault is the
  only way in or out.
- A vault with `ssh-timeout 0` never contributes keys.

## Security

- **Encryption**: AES-256-GCM with Argon2id key derivation
- **Atomic writes**: vault is written to a temp file then renamed, preventing
  corruption
- **File locking**: advisory flock prevents concurrent writes
- **Memory zeroing**: passwords, keys, and plaintext are zeroed after use
- **Clipboard clearing**: passwords auto-clear from clipboard after 30 seconds
- **Agent security**: Unix sockets with 0600 permissions, passwords zeroed on
  expiry and lock; SSH keys are served as signatures only, each one
  approved by you by default
