# hudo

A `sudo` replacement for autonomous agent programs running on Linux. Instead of granting agents unrestricted root access, `hudo` requires a human to approve each privileged command via a PIN sent to a webhook (e.g. Telegram).

## How it works

```
Agent                    hudo              Human (Telegram)
  │                          │                          │
  ├─ request "rm -rf /old" ─>│                          │
  │                          ├─ generate PIN ──────────>│
  │<── OK ───────────────────┤   send notification      │
  │                          │                          │
  │   ask human for PIN ─────────────────────────────>  │
  │<─ PIN: 847291 ─────────────────────────────────────<│
  │                          │                          │
  ├─ exec "rm -rf /old"      │                          │
  │   --pin 847291 ─────────>│                          │
  │                          ├─ verify PIN + HMAC       │
  │                          ├─ run as root             │
  │<── stdout/stderr/exit ───┤                          │
```

1. Agent calls `hudo request <command>` — a 6-digit PIN and HMAC signature are generated, stored locally, and sent to your webhook.
2. Human sees the notification, reads the PIN, passes it to the agent.
3. Agent calls `hudo exec <command> --pin <PIN>` — PIN and HMAC are verified, command runs as root, output is passed through.

PIN is **one-time** and expires after a configurable TTL (default 5 minutes).

## Security model

- **HMAC-SHA256** ties the PIN to the exact command — a PIN for `ls` cannot be used to run `rm`.
- **setuid root binary** — the binary runs as root; the config file (`hmac_secret`, `webhook_url`) is readable only by root, not by the agent process.
- **Atomic one-time use** — PIN is deleted from the store in the same transaction that reads it; no replay window.
- **Clean environment** — child process receives only `PATH=/usr/local/sbin:...`; `LD_PRELOAD` and similar variables are stripped.

## Installation

Requires root. Replace `OWNER` with the actual GitHub username before running.

```sh
curl -fsSL https://raw.githubusercontent.com/mitrii/hudo/main/install.sh | sudo sh
```

Or clone and run manually:

```sh
git clone https://github.com/mitrii/hudo
sudo ./hudo/install.sh
```

The script will:
- Download the latest release binary from GitHub Releases
- Install it to `/usr/local/bin/hudo` with setuid root
- Create `/etc/hudo/` and `/var/lib/hudo/` with `0700 root` permissions
- Generate a random `hmac_secret` and ask for your `webhook_url`
- Write `/etc/hudo/config.yaml` (readable only by root)

To install a specific version:

```sh
sudo ./install.sh v1.2.0
```

## Configuration

`/etc/hudo/config.yaml` — owner `root:root`, permissions `0600`:

```yaml
hmac_secret: "..."        # HMAC signing secret — generated automatically by install.sh
webhook_url: "..."        # URL to send PIN notifications to
store_path: "/var/lib/hudo/pending.db"
pin_ttl_seconds: 300
```

All fields can be overridden with environment variables:

| Variable | Config key |
|---|---|
| `HUDO_HMAC_SECRET` | `hmac_secret` |
| `HUDO_WEBHOOK_URL` | `webhook_url` |
| `HUDO_STORE_PATH` | `store_path` |
| `HUDO_PIN_TTL_SECONDS` | `pin_ttl_seconds` |

### Telegram webhook URL

```
https://api.telegram.org/bot<TOKEN>/sendMessage?chat_id=<CHAT_ID>&text={text}
```

The `{text}` placeholder is replaced with the URL-encoded notification message. Any webhook that accepts a GET request with the message in the URL works the same way.

For plain POST webhooks (without `{text}`), the message is sent as `application/x-www-form-urlencoded` with field `text`.

## Usage

```sh
# Step 1 — agent requests permission
hudo request "systemctl restart nginx"
# prints: OK

# Step 2 — human receives notification and provides PIN to the agent

# Step 3 — agent executes with PIN
hudo exec "systemctl restart nginx" --pin 847291
# stdout/stderr of the command are passed through; exit code is propagated
```

## Building from source

Requires Go 1.26+. Linux is required for the privilege escalation path; `go build ./...` works on macOS via a stub.

```sh
git clone https://github.com/mitrii/hudo
cd hudo
go build -o hudo .

# Install manually
sudo chown root:root hudo
sudo chmod 4755 hudo        # setuid root
sudo mv hudo /usr/local/bin/
```

## Releasing

Push a semver tag — CI builds the binary and creates a GitHub Release automatically:

```sh
git tag v1.0.0
git push origin v1.0.0
```

## Requirements

- Linux (privilege escalation uses `setuid`)
- `curl`, `openssl` — for `install.sh`
- Webhook endpoint reachable from the server (HTTPS recommended)
