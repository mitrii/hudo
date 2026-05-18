#!/usr/bin/env sh
# install.sh — install hudo on Linux
# Usage: sudo ./install.sh [version] [webhook_url]
#   version:     tag name, e.g. v1.0.0 (default: latest release)
#   webhook_url: e.g. https://api.telegram.org/bot<TOKEN>/sendMessage?chat_id=<ID>&text={text}
#
# When piped through curl | sudo sh, pass arguments via env:
#   curl -fsSL https://... | sudo WEBHOOK_URL="https://..." sh
set -eu

REPO="mitrii/hudo"
INSTALL_PATH="/usr/local/bin/hudo"
CONFIG_DIR="/etc/hudo"
STORE_DIR="/var/lib/hudo"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"

# ── helpers ──────────────────────────────────────────────────────────────────

die() {
  echo "error: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed"
}

# ── preflight ─────────────────────────────────────────────────────────────────

[ "$(id -u)" -eq 0 ] || die "run as root: sudo $0"
[ "$(uname -s)" = "Linux" ] || die "Linux only"

need curl
need openssl

# ── detect architecture ───────────────────────────────────────────────────────

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)              BINARY_SUFFIX="amd64" ;;
  aarch64 | arm64)     BINARY_SUFFIX="arm64" ;;
  armv7l | armv7)      BINARY_SUFFIX="armv7" ;;
  armv6l | armv6)      BINARY_SUFFIX="armv6" ;;
  i386 | i486 | i586 | i686) BINARY_SUFFIX="386" ;;
  *)                   die "unsupported architecture: ${ARCH}" ;;
esac

BINARY_NAME="hudo-linux-${BINARY_SUFFIX}"
echo "Detected architecture: ${ARCH} → ${BINARY_NAME}"

# ── resolve version ───────────────────────────────────────────────────────────

VERSION="${1:-}"

if [ -z "$VERSION" ]; then
  echo "Fetching latest release..."
  VERSION=$(
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/'
  )
  [ -n "$VERSION" ] || die "could not determine latest version (is ${REPO} published?)"
fi

echo "Installing hudo ${VERSION}..."

# ── download ──────────────────────────────────────────────────────────────────

TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"

echo "Downloading ${DOWNLOAD_URL}..."
curl -fSL --progress-bar "$DOWNLOAD_URL" -o "$TMP" \
  || die "download failed — check that release ${VERSION} exists and has asset ${BINARY_NAME}"

# ── install binary ────────────────────────────────────────────────────────────

install -o root -g root -m 0755 "$TMP" "$INSTALL_PATH"
chmod u+s "$INSTALL_PATH"   # setuid root

echo "Binary installed: ${INSTALL_PATH} (setuid root)"

# ── create directories ────────────────────────────────────────────────────────

install -d -o root -g root -m 0700 "$CONFIG_DIR"
install -d -o root -g root -m 0700 "$STORE_DIR"

# ── config ────────────────────────────────────────────────────────────────────

if [ -f "$CONFIG_FILE" ]; then
  echo "Config already exists, skipping: ${CONFIG_FILE}"
else
  HMAC_SECRET=$(openssl rand -hex 32)

  # Resolve webhook URL: arg $2 → env WEBHOOK_URL → interactive prompt
  WEBHOOK_URL="${2:-${WEBHOOK_URL:-}}"

  if [ -z "$WEBHOOK_URL" ]; then
    # Interactive prompt — requires a usable terminal.
    # When piped through curl | sudo sh, pass via env instead:
    #   curl -fsSL https://... | sudo WEBHOOK_URL="https://..." sh
    if [ -t 0 ] || { TTY_FD=$(command -v /dev/tty 2>/dev/null) && [ -c /dev/tty ]; }; then
      printf "Webhook URL (e.g. https://api.telegram.org/bot<TOKEN>/sendMessage?chat_id=<ID>&text={text}): "
      read -r WEBHOOK_URL </dev/tty || true
    fi
  fi

  [ -n "$WEBHOOK_URL" ] || die "webhook_url is required — pass it as: sudo WEBHOOK_URL=\"<url>\" sh install.sh"

  cat > "$CONFIG_FILE" <<EOF
hmac_secret: "${HMAC_SECRET}"
webhook_url: "${WEBHOOK_URL}"
store_path: "${STORE_DIR}/pending.db"
pin_ttl_seconds: 300
EOF

  chmod 600 "$CONFIG_FILE"
  echo "Config written: ${CONFIG_FILE}"
fi

# ── done ──────────────────────────────────────────────────────────────────────

echo ""
echo "Installation complete."
echo "  Binary:  ${INSTALL_PATH}"
echo "  Config:  ${CONFIG_FILE}"
echo "  Store:   ${STORE_DIR}/pending.db"
echo ""
echo "Edit ${CONFIG_FILE} if you need to change webhook_url or pin_ttl_seconds."
