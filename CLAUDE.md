# Passman

@AGENTS.md

## Development Rules

- Use TDD (Red → Code → Green) for all changes
- Run `go test ./...` to verify before considering work done
- No dead code — remove unused functions, types, and imports
- No code duplication — extract helpers when patterns repeat
- Use `crypto.FastKDFParams()` in tests for speed
- Entry model is simple: `{Name, Value}` — no username, notes, tags, or TOTP
- Vault version is 2 — do not regress to version 1

## Security

- Zero passwords/keys after use with `vault.ZeroBytes()`
- Never log or include secrets in error messages
- Validate all external input (files, config values)
- Shell-escape values in export output
- Agent socket must use umask protection (not post-creation chmod)
