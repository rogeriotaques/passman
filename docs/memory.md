# Work Log

## 2026-11-18: SSH agent + SSH_ASKPASS (0.4.0)

- **Why**: a security review found three unencrypted SSH private keys in `~/.ssh` (GitHub + bastions, two AWS `.pem`). Instead of 1Password's SSH agent, passman now plays that role: keys live in the vault, ssh gets signatures only.
- **New packages**:
  - `internal/sshagent`: `Keyring` implements `agent.ExtendedAgent` (x/crypto). Keys grouped by source vault with TTL; `Sign`/`SignWithFlags` (rsa-sha2-256/512) gated by a `Confirmer`; `Add`/`Remove`/`Lock` rejected (`ErrUnsupported`); `ServeConn` resolves the peer pid/name (`peer_darwin.go` LOCAL_PEERPID, `peer_linux.go` SO_PEERCRED) for the confirmation text; `OnEmpty` hook for on-demand unlock; exported `ParseKey`.
  - `internal/prompt`: `Prompter` (Confirm/Secret/Text) + `Mock`. macOS uses `/usr/bin/osascript` with the message passed as an **argument** (`on run argv`), not interpolated — verified `system attribute` mangles UTF-8 and string interpolation would be injectable. Other OSes delegate to `ssh-askpass`.
- **Agent** (`internal/agent/server.go`): `EnableSSH(SSHConfig)` adds a second listener. `Store` synchronously loads the vault's keys (skipped when file mtime/size and password are unchanged — stat + constant-time compare), reading `ssh-timeout`/`ssh-confirm` from the vault's `config.json`. `Lock`/expiry/`Stop` clear keys. On-demand unlock (`UnlockVault` + `PromptPassword`) loads keys **without** caching the master password; cooldown 10 s doubling per cancelled prompt up to 15 min (`unlockFailures`), reset on success; `ssh-unlock-prompt off` disables it. `idle()` keeps the agent alive when on-demand unlock is configured. `EnsureAgent(sock, sshSock, sshVault)` passes `--ssh-socket/--ssh-vault`, now `Setsid` so closing the terminal does not kill the agent; `_agent` stops on SIGINT/SIGTERM/SIGHUP.
- **Config**: `ssh-timeout` (minutes, default 480, 0 disables), `ssh-confirm` (on/off, default on), `ssh-unlock-prompt` (on/off, default on); `SSHKeyTTL()`, `SSHConfirmEnabled()`; shared `onOff`/`parseOnOff`/`parseMinutes` helpers.
- **Commands** (`cmd/ssh*.go`): `ssh add <file> [--name]` (64 KiB limit, decrypts with prompted passphrase, re-marshals unencrypted via `ssh.MarshalPrivateKey`, prints fingerprint + public key), `ssh list`, `ssh env` (exports + writes `~/.passman/askpass` wrapper, 0700, atomic; `SSH_ASKPASS_REQUIRE=force` because ssh ignores askpass without `DISPLAY`), `ssh askpass` (confirm/none/secret modes; vault lookup `ssh/passphrase/<file>` for OpenSSH prompt wordings incl. ssh-keygen's `"..."` and ssh-add's `(will confirm each use)`; never answers "Bad passphrase" retries from the vault; `attachTTY()` reopens `app.TTYPath` when stdin is redirected).
- **App**: `SSHSocketPath`, `SSHUnlockVault` (always the default vault), `AskpassPath`, `Prompter`, `GUIPassword`, `TTYPath`. `readPassword`: terminal → dialog (when `GUIPassword`) → stdin line. `saveAndSync` re-stores the password so a running agent reloads keys right after `ssh add`/`remove`.
- **Tests**: `internal/sshagent/keyring_test.go`, `internal/agent/ssh_test.go`, `internal/prompt/prompt_test.go` (fake osascript), `cmd/ssh_test.go`, config tests. Lesson: `app.GUIPassword` leaked between tests and silently created passwordless vaults — `setupTestApp` now resets it and sets `TTYPath` to `/dev/null`.
- **Verified manually** (isolated `HOME`): `ssh-add -l` lists vault keys; `ssh-keygen -Y sign` through the agent for ed25519 and RSA (`rsa-sha2-512`); `ssh-add -D` refused; `ssh-keygen -y` on an encrypted file key answered by `ssh askpass` from the vault.
- **Docs**: README, `docs/DOCS.md` (ssh commands, config keys, "SSH agent" section), knowledge files. `VERSION` 0.4.0.
- **Git sync hardening** (same day, while backing up the real vault): `Init` uses `git init -b main` and writes `.gitignore` (`*.lock`, `*.tmp`); `gitCmd` passes `-c commit.gpgsign=false` and `GIT_TERMINAL_PROMPT=0` (the user's global `commit.gpgsign=true` would have made background sync prompt for GPG); `Sync` skips `pull --rebase` when `ls-remote --exit-code` says the remote has no `main` yet (exit 2) so the first push to an empty repo works; `vault config git <url>` now initialises the repo for an existing vault. Tests isolate git with `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_NOSYSTEM` and use bare `-b main` remotes.
- **Approval grace** (user got two dialogs for one `vault sync`): `LoadOptions.ConfirmGrace` / config `ssh-confirm-grace` (seconds, default 60). `Keyring.approved()` remembers an Allow per `approvalKey{fingerprint, peer executable path}`; unknown peers and grace 0 always prompt; Deny deletes the approval; `Unload`/`Expire`/`Clear` drop approvals of the affected keys. `Sync` now pushes optimistically and only pulls on a rejected push (one connection → one dialog); `remoteHasBranch` removed.
- **Lifecycle bugs** (found restarting the real agent): (1) `Start` returned when the control listener closed, so the process exited mid-`Stop` and left a stale `ssh-agent.sock` (ssh then gets "connection refused"); now `Stop` uses `sync.Once` and closes a `stopped` channel that `Start` and `Done()` wait on. (2) Nothing restarted the agent after a reboot until a vault was unlocked; `ssh env` now calls `ensureAgent` (warning only on failure). Tests: `TestServer_StartReturnsOnlyAfterCleanup`, `TestSSHEnv_StartsAgent`, `TestSSHEnv_AgentStartFailureIsOnlyAWarning`. Suite clean under `-race`.
- **Gotcha**: `ssh-add -l` against an agent with no keys loaded triggers the on-demand unlock dialog (that is the design: listing is how ssh discovers keys). Never use it as a passive health check.
- **Not done / ideas**: list public keys while locked (cache `.pub` data outside the vault) and unlock on the first sign instead of the first list; Touch ID instead of the Deny/Allow dialog (needs cgo + LocalAuthentication); `passman lock` on screen lock; per-key confirm setting; export the public key of a stored key (`ssh pubkey`).

## 2026-10-28: Show version in the help banner

- **Change**: split `banner` into `bannerArt` + `bannerText()` in `cmd/root.go`. The tagline now ends with `• v<version>`, e.g. `Manage secrets from your terminal • v0.3.0`.
- **Wiring**: `SetVersion` also refreshes `rootCmd.Long` — Cobra copied it at init, the same trap as `rootCmd.Version`.
- **Tests**: `TestSetVersion_UpdatesBanner`.
- **Deployed**: rebuilt and installed `/usr/local/bin/passman` (0.3.0).

## 2026-10-28: Group CLI by noun (0.3.0)

- **Change**: top-level commands grouped by noun. New tree:
  - `secret`: `add`, `get`, `list`, `remove`, `export`, `import`
  - `vault`: `init`, `list`, `passwd`, `config`, `sync`, `destroy`
  - root: `generate`, `lock`, `upgrade` (+ hidden `_agent`)
- **Renames**: `rm` → `remove`; `vaults` → `vault list`. No aliases or shims (single user).
- **Files**: new `cmd/secret.go`, `cmd/vault.go`; `cmd/rm.go` → `cmd/remove.go`; `cmd/vaults.go` → `cmd/vault_list.go`; registrations moved from `rootCmd` to the group commands.
- **Tests**: every `runCmd` call updated; new `TestVaultList`. Full suite green.
- **Docs**: `README.md`, `docs/DOCS.md` (rewritten command sections), `docs/knowledge/commands.md`.
- **Version**: bumped minor to `0.3.0`; rebuilt and installed `/usr/local/bin/passman`.

## 2026-10-28: Enforce semver on `VERSION`

- **Change**: added `cmd/version.go` with a strict semver pattern (`MAJOR.MINOR.PATCH`, optional pre-release/build, no leading zeros). `version` and `SetVersion` moved there from `root.go`.
- **Behaviour**:
  - `SetVersion` returns an error for empty/invalid values; `main.go` prints it and exits 1.
  - `fetchLatestVersion` validates the clone's `VERSION` before `upgrade` builds.
  - `TestRepoVersionFile_IsSemver` fails CI if the committed `VERSION` is malformed.
- **Tests**: `TestSetVersion_AcceptsSemver`, `TestSetVersion_RejectsInvalid`, `TestFetchLatestVersion_Invalid`, `TestRepoVersionFile_IsSemver`.
- **Deployed**: rebuilt `/usr/local/bin/passman`; reports `0.2.0`.
- **Files changed**: `cmd/version.go`, `cmd/root.go`, `cmd/upgrade.go`, `main.go`, `cmd/cmd_test.go`, `cmd/upgrade_test.go`, `docs/DOCS.md`, `docs/knowledge/tech-stack.md`

## 2026-10-28: Switch to semver (0.2.0)

- **Change**: replaced the commit-hash version with semver. `VERSION` at the repo root is the single source of truth (`0.2.0`).
- **Wiring**: `main.go` embeds `VERSION` with `//go:embed` and calls `cmd.SetVersion`, which updates both the package var and `rootCmd.Version` (Cobra copies the latter at init). `version = "dev"` remains only as a fallback for tests/tools.
- **upgrade**: `fetchLatestCommitHash` → `fetchLatestVersion`; it reads `VERSION` from the shallow clone. The build no longer injects `-ldflags -X ...version=<hash>` because the clone embeds its own `VERSION`. Output now shows semver.
- **Tests**: `TestFetchLatestVersion`, `TestFetchLatestVersion_MissingFile`, `TestSetVersion_UpdatesReportedVersion`, `TestSetVersion_EmptyKeepsCurrent`; upgrade tests now use a fake repo with `VERSION`.
- **Deployed**: rebuilt `/usr/local/bin/passman`; `passman --version` reports `0.2.0`.
- **Files changed**: `VERSION`, `main.go`, `cmd/root.go`, `cmd/upgrade.go`, `cmd/upgrade_test.go`, `cmd/cmd_test.go`, `README.md`, `docs/DOCS.md`, `docs/knowledge/tech-stack.md`

## 2026-10-28: Verify `add` on a missing vault leaves no lock

- **Report**: a failed `add` on a nonexistent vault supposedly locked the vault and blocked `init`.
- **Finding**: the source already returns `ErrVaultNotFound` before locking (`Store.Exists()` check in `Load` and in `loadVault`), so it creates no directory and no `.lock` file. The installed binary (`/usr/local/bin/passman`, commit 4c0534f, May 26) still prompts for the password and leaves `<vault>.enc.lock` plus the vault directory, but `init` still succeeded in the repro.
- **Change**: added `TestAdd_NonexistentVault_CreatesNoFiles` to pin "no directory, no lock file" after a failed add.
- **Deployed**: rebuilt `/usr/local/bin/passman` from the current tree (version `e34c0b3`), replacing the stale May 26 build.
- **Files changed**: `cmd/cmd_test.go`

## 2026-10-28: Stop tests from spawning agent processes

- **Bug**: `cmd` tests that touched a non-empty `SocketPath` reached `cacheInAgent` → `agent.EnsureAgent`, which re-executes `os.Executable()`. Under `go test` that is the test binary, and Go's flag parser stops at the first non-flag arg (`_agent`), so the child ran the **entire test suite again** — each pass spawning more children, exponentially.
- **Fix**:
  - Added `App.AutoStartAgent` (default `true` for the real app); `cacheInAgent` returns before spawning when it is false.
  - `setupTestApp` and the destroy tests set it to `false`.
  - Added `cmd/main_test.go` `TestMain`: if the test binary is invoked with `_agent`, it prints "refusing to start agent from test binary" and exits 1.
  - Added `ensureAgent` package seam so tests can assert zero spawn attempts.
- **Tests**: `TestCacheInAgent_AutoStartDisabled_DoesNotSpawn`, `TestCacheInAgent_AutoStartEnabled_SpawnsAgent`, `TestTestBinary_RefusesAgentInvocation`.
- **Files changed**: `cmd/root.go`, `cmd/input.go`, `cmd/main_test.go`, `cmd/cmd_test.go`, `docs/knowledge/tests.md`

## 2026-05-26: Add `passman upgrade` command

- **Feature**: Added `upgrade` command to auto-upgrade passman by cloning the repo, compiling, and replacing the current binary.
- **Files changed**: `cmd/root.go`, `cmd/upgrade.go`, `cmd/upgrade_test.go`, `docs/DOCS.md`, `docs/knowledge/commands.md`
- **Details**:
  - Added `version` variable (`"dev"` by default) to `cmd/root.go`, wired into `rootCmd.Version` for `--version` flag.
  - `upgrade` clones the `master` branch (shallow), reads the latest commit hash via `git rev-parse --short HEAD`, builds with `-ldflags` to bake the hash into the new binary, and atomically replaces the current executable.
  - Includes cross-device safe binary replacement fallback (copy + rename).
  - Tests cover: commit hash parsing, already-up-to-date, missing git/go errors, and end-to-end successful upgrade against a local `file://` repo.

## 2026-05-26: Fix wrong-password agent caching

- **Bug**: When user entered a wrong password, `getPassword()` cached it in the agent immediately. On the next command, the wrong password was retrieved from agent without prompting.
- **Fix**: Moved `cacheInAgent(password)` from `getPassword()` to `loadVault()` so passwords are only cached after successful decryption.
- **Files changed**: `cmd/input.go`, `cmd/root.go`, `cmd/cmd_test.go`

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
