# AGENTS.md

## Project purpose

`hudo` is a Go CLI tool that acts as a `sudo` replacement for autonomous agent programs running on Linux. It handles privilege escalation via human-in-the-loop PIN approval sent through a configurable webhook (e.g. Telegram Bot API).

## Module

- Module: `hudo`
- Go version: 1.26
- Target platform: **Linux only** — privilege escalation code (`syscall.Setuid/Setgid`) is in `cmd/exec_linux.go`. Non-Linux stub is in `cmd/exec_other.go` so `go build ./...` works on macOS.

## Commands

```sh
go build ./...          # compile
go test ./...           # run all tests
go vet ./...            # static analysis
golangci-lint run ./... # lint (strict, see .golangci.yml)
```

Run a single package: `go test ./internal/store/...`

## Architecture

```
main.go                        # cobra root; adds RequestCmd and ExecCmd
cmd/
  request.go                   # "request <cmd>": generate PIN, HMAC, notify, save to store
  exec.go                      # "exec <cmd> --pin <PIN>": verify, consume, run
  exec_linux.go                # runPrivileged(): Setgid(0)+Setuid(0), clean env, exec
  exec_other.go                # stub for non-Linux
internal/
  config/config.go             # load /etc/hudo/config.yaml + HUDO_* env
  store/store.go               # bbolt: Save / Consume (atomic get+delete) / Purge
  notify/notify.go             # HTTP POST or GET with {text} placeholder
config.yaml.example
.github/workflows/ci.yml       # CI: vet → golangci-lint → test → build; release on v* tags
install.sh                     # install from GitHub Releases (requires root)
```

### Protocol

1. Agent calls `hudo request <cmd>` → PIN generated, `HMAC-SHA256(secret, cmd+pin)` computed, stored in bbolt keyed by HMAC, webhook fired. Prints `OK` on success.
2. Human sees notification (Telegram etc.), reads PIN.
3. Agent calls `hudo exec <cmd> --pin <PIN>` → HMAC recomputed, store entry consumed atomically, command run as root, stdout/stderr passed through, exit code propagated.

### HMAC

`HMAC-SHA256(hmac_secret, command + pin)` — no timestamp. PIN is one-time (deleted on first successful `exec`). TTL enforced via `expires_at` stored in bbolt value.

## Config

File: `/etc/hudo/config.yaml`, owner `root:root`, permissions `0600`.

```yaml
hmac_secret: "..."          # or HUDO_HMAC_SECRET env
webhook_url: "..."          # or HUDO_WEBHOOK_URL env
store_path: "/var/lib/hudo/pending.db"
pin_ttl_seconds: 300
```

`webhook_url` with `{text}` placeholder → GET request (Telegram Bot API compatible).
Without placeholder → POST `application/x-www-form-urlencoded`.

**Viper quirk**: `AutomaticEnv` alone does not override keys when no config file is present. Each key must also be registered with `v.BindEnv("key", "ENV_VAR")` — see `internal/config/config.go`. Adding new config fields requires a matching `BindEnv` call.

## Installation (Linux)

```sh
go build -o hudo .
sudo chown root:root hudo
sudo chmod 4755 hudo           # setuid root
sudo mv hudo /usr/local/bin/
sudo mkdir -p /etc/hudo /var/lib/hudo
sudo chmod 700 /etc/hudo /var/lib/hudo
sudo cp config.yaml.example /etc/hudo/config.yaml
sudo chmod 600 /etc/hudo/config.yaml
# edit /etc/hudo/config.yaml: set hmac_secret and webhook_url
```

## Security notes

- **Environment sanitization**: `exec_linux.go` passes only `PATH=/usr/local/sbin:...` to the child process — `LD_PRELOAD`, `LD_LIBRARY_PATH` etc. are stripped.
- **Atomic one-time use**: bbolt write transaction does get+delete atomically — no replay or race window.
- **setuid bitmask**: binary must be owned by root with setuid bit; store directory must be `0700 root` so agent cannot read pending PINs from the db file directly.
- **Webhook must use HTTPS**: Go's `net/http` verifies TLS by default; do not use plain HTTP webhook URLs.
- **Command mismatch check**: `exec.go` verifies `entry.Command == command` after HMAC lookup as a defence-in-depth measure.

## Gotchas

- **bbolt transaction rollback**: returning any non-nil error from a `db.Update` callback rolls back the entire transaction, including prior `Delete` calls. To signal a semantic error (e.g. `ErrExpired`) while still committing a delete, use an outer boolean flag and return `nil` from the callback — see `store.Consume`.
- **cobra stateful flags**: `pinFlag` is a package-level var mutated by cobra. Tests must reset it (`pinFlag = ""`) and build a fresh `cobra.Command` per test case to avoid cross-test pollution — see `cmd/exec_test.go:newRoot()`.
- **Lint version**: CI uses `golangci-lint v2.12.2` with a v2 config (`version: "2"` in `.golangci.yml`). Running v1 locally against this config will fail with a version mismatch error.
