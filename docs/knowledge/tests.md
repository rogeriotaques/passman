# Tests

## Running Tests

```bash
# All tests
go test ./...

# Verbose
go test ./... -v

# Single package
go test ./internal/vault/ -v

# Single test
go test ./cmd/ -run TestAdd_And_Get -v
```

## Test Strategy

- **Unit tests**: Each internal package has its own `_test.go` files
- **Integration tests**: `cmd/cmd_test.go` tests full command flows end-to-end
- **TDD**: Tests are written before implementation (Red → Code → Green)

## Test Helpers

### `cmd/cmd_test.go`

- `setupTestApp(t, input)` — configure test app with mock I/O, fast KDF params
- `setInput(input)` — set stdin for subsequent commands
- `resetFlags()` — reset all cobra flags between test runs
- `runCmd(args...)` — execute a command and return error
- `testSocketPath(t)` — create a temp Unix socket path
- `startTestAgent(t, sock)` — start a test agent server

### Fast KDF Params

Tests use `crypto.FastKDFParams()` (time=1, memory=1024, threads=1) to avoid
the ~300ms real Argon2id cost on each encrypt/decrypt operation.

## Test Coverage by Package

| Package | Tests |
|---------|-------|
| `internal/vault` | vault_test.go, storage_test.go |
| `internal/crypto` | crypto_test.go |
| `internal/importer` | env_test.go, csv_test.go |
| `internal/generate` | generate_test.go |
| `internal/sync` | config_test.go, git_test.go |
| `internal/agent` | agent_test.go |
| `internal/selector` | selector_test.go |
| `cmd` | cmd_test.go (integration) |

## Key Testing Patterns

1. All vault operations use `t.TempDir()` for isolation
2. Password/value input is simulated via `strings.NewReader`
3. Clipboard uses `MockClipboard` (no system clipboard interaction)
4. Agent tests use a real Unix socket in `/tmp`
5. Git tests use `git init` in temp dirs (no remote)
