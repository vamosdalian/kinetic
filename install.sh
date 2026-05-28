#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# Kinetic installer / upgrader
# Usage:
#   curl -sSL https://raw.githubusercontent.com/vamosdalian/kinetic/master/install.sh | bash
#   curl -sSL .../install.sh | bash -s -- --mode worker --controller-url http://host:9898
#
# Options (passed after --):
#   --mode <controller|worker>   default: controller
#   --controller-url <url>       required when --mode worker
#   --version <vX.Y.Z>           pin a specific release (default: latest)
#   --install-dir <path>         default: /usr/local/bin
# ---------------------------------------------------------------------------

REPO="vamosdalian/kinetic"
SERVICE_NAME="kinetic"
SERVICE_USER="$(id -un)"
SERVICE_GROUP="$(id -gn)"

INSTALL_DIR="/usr/local/bin"
MODE="controller"
CONTROLLER_URL=""
PINNED_VERSION=""

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)            MODE="$2";            shift 2 ;;
    --controller-url)  CONTROLLER_URL="$2";  shift 2 ;;
    --version)         PINNED_VERSION="$2";  shift 2 ;;
    --install-dir)     INSTALL_DIR="$2";     shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ "$MODE" == "worker" && -z "$CONTROLLER_URL" ]]; then
  echo "Error: --controller-url is required when --mode is worker"
  exit 1
fi

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()    { echo "[kinetic] $*"; }
success() { echo "[kinetic] OK: $*"; }
error()   { echo "[kinetic] Error: $*" >&2; exit 1; }

need_cmd() { command -v "$1" &>/dev/null || error "required command not found: $1"; }
need_cmd curl
need_cmd tar

# ---------------------------------------------------------------------------
# Detect platform
# ---------------------------------------------------------------------------
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)  OS_NAME="linux" ;;
  Darwin) OS_NAME="darwin" ;;
  *)      error "Unsupported OS: $OS" ;;
esac

case "$ARCH" in
  x86_64)          ARCH_NAME="x86_64" ;;
  aarch64 | arm64) ARCH_NAME="arm64" ;;
  *)               error "Unsupported architecture: $ARCH" ;;
esac

# ---------------------------------------------------------------------------
# Resolve version
# ---------------------------------------------------------------------------
if [[ -n "$PINNED_VERSION" ]]; then
  VERSION="$PINNED_VERSION"
else
  info "Fetching latest release version..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' | head -1 | cut -d'"' -f4)"
  [[ -n "$VERSION" ]] || error "Failed to fetch latest version from GitHub"
fi

info "Version : $VERSION"
info "Platform: $OS_NAME / $ARCH_NAME"
info "Mode    : $MODE"
info "User    : $SERVICE_USER"

# ---------------------------------------------------------------------------
# Detect upgrade vs fresh install
# ---------------------------------------------------------------------------
UPGRADE=false
if [[ -f "$INSTALL_DIR/kinetic" ]]; then
  CURRENT_VERSION="$("$INSTALL_DIR/kinetic" --version 2>/dev/null | awk '{print $2}' || echo "unknown")"
  if [[ "$CURRENT_VERSION" == "$VERSION" ]]; then
    info "kinetic $VERSION is already installed. Nothing to do."
    exit 0
  fi
  info "Upgrading $CURRENT_VERSION -> $VERSION"
  UPGRADE=true
else
  info "Fresh install"
fi

# ---------------------------------------------------------------------------
# Download & verify
# ---------------------------------------------------------------------------
ASSET="kinetic_${VERSION}_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading $ASSET..."
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET"

info "Verifying checksum..."
curl -fsSL "$CHECKSUM_URL" -o "$TMP_DIR/checksums.txt"
(cd "$TMP_DIR" && grep "$ASSET" checksums.txt | sha256sum --check --status) \
  || error "Checksum verification failed"

info "Extracting..."
tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"

# ---------------------------------------------------------------------------
# Stop service before replacing binary (upgrade only)
# ---------------------------------------------------------------------------
if $UPGRADE && systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
  info "Stopping $SERVICE_NAME service for upgrade..."
  sudo systemctl stop "$SERVICE_NAME"
fi

# ---------------------------------------------------------------------------
# Install binary
# ---------------------------------------------------------------------------
sudo install -m 755 "$TMP_DIR/kinetic" "$INSTALL_DIR/kinetic"
success "Binary installed at $INSTALL_DIR/kinetic"

# On upgrade: just reload and restart, config/service already exist
if $UPGRADE; then
  sudo systemctl daemon-reload
  sudo systemctl start "$SERVICE_NAME"
  success "kinetic upgraded to $VERSION and restarted"
  systemctl status "$SERVICE_NAME" --no-pager || true
  exit 0
fi

# ---------------------------------------------------------------------------
# Create systemd service (Linux only)
# Config and data dirs (~/.kinetic/) are managed by kinetic itself on first start.
# ---------------------------------------------------------------------------
if [[ "$OS_NAME" != "linux" ]]; then
  success "Non-Linux install complete. Run: kinetic"
  exit 0
fi

SYSTEMD_UNIT="/etc/systemd/system/$SERVICE_NAME.service"
info "Writing systemd unit to $SYSTEMD_UNIT..."

EXEC_START="$INSTALL_DIR/kinetic"
EXTRA_ENV=""
if [[ "$MODE" == "worker" ]]; then
  EXEC_START="$EXEC_START --mode worker"
  EXTRA_ENV="Environment=KINETIC_WORKER_CONTROLLER_URL=$CONTROLLER_URL"
fi

sudo tee "$SYSTEMD_UNIT" > /dev/null <<EOF
[Unit]
Description=Kinetic workflow engine
Documentation=https://github.com/$REPO
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
ExecStart=$EXEC_START
WorkingDirectory=$(eval echo "~$SERVICE_USER")
$EXTRA_ENV
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable "$SERVICE_NAME"
sudo systemctl start "$SERVICE_NAME"

CONFIG_FILE="$(eval echo "~$SERVICE_USER")/.kinetic/config.yml"

success "kinetic $VERSION installed and started"
echo ""
echo "  Status : systemctl status $SERVICE_NAME"
echo "  Logs   : journalctl -u $SERVICE_NAME -f"
echo "  Config : $CONFIG_FILE"
if [[ "$MODE" == "controller" ]]; then
  echo "  Web UI : http://$(hostname -I | awk '{print $1}'):9898"
  echo ""
  echo "  Default credentials: kinetic / kinetic"
  echo "  Change them in $CONFIG_FILE"
fi
echo ""
