#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SERVICE_NAME="${SERVICE_NAME:-music-server}"
RUN_USER="${RUN_USER:-$(id -un)}"
WORK_DIR="${WORK_DIR:-/opt/music-server}"
WEBUI_DIR="${WEBUI_DIR:-$REPO_ROOT/player}"
MUSIC_ROOT="${MUSIC_ROOT:-$WORK_DIR/data}"
PORT="${PORT:-8000}"

BIN_DIR="$WORK_DIR/bin"
SERVICE_TEMPLATE="$SCRIPT_DIR/music-server.service.tmpl"
CONFIG_TEMPLATE="$SCRIPT_DIR/music-server.yaml.tmpl"
SERVICE_TARGET="/etc/systemd/system/$SERVICE_NAME.service"
SERVICE_RENDERED="$(mktemp)"
CONFIG_RENDERED="$(mktemp)"

cleanup() {
  rm -f "$SERVICE_RENDERED" "$CONFIG_RENDERED"
}
trap cleanup EXIT

require_root() {
  if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
    echo "error: this script must run as root (use sudo)." >&2
    exit 1
  fi
}

render_template() {
  local src="$1"
  local dst="$2"

  sed \
    -e "s|{{RUN_USER}}|$RUN_USER|g" \
    -e "s|{{WORK_DIR}}|$WORK_DIR|g" \
    -e "s|{{MUSIC_ROOT}}|$MUSIC_ROOT|g" \
    -e "s|{{PORT}}|$PORT|g" \
	-e "s|{{WEBUI_DIR}}|$WEBUI_DIR|g" \
    "$src" > "$dst"
}

build_server() {
  echo "[1/4] build server binary"
  mkdir -p "$BIN_DIR"
  (cd "$REPO_ROOT/server" && go build -o "$BIN_DIR/music-server" .)
  chmod 0755 "$BIN_DIR/music-server"
}

install_config() {
  echo "[2/4] install config"
  mkdir -p "$WORK_DIR"
  render_template "$CONFIG_TEMPLATE" "$CONFIG_RENDERED"
  install -m 0644 "$CONFIG_RENDERED" "$WORK_DIR/music-server.yaml"
}

install_service() {
  echo "[3/4] install systemd service"
  render_template "$SERVICE_TEMPLATE" "$SERVICE_RENDERED"
  install -m 0644 "$SERVICE_RENDERED" "$SERVICE_TARGET"
}

restart_service() {
  echo "[4/4] reload and restart service"
  systemctl daemon-reload
  systemctl enable --now "$SERVICE_NAME"
  systemctl restart "$SERVICE_NAME"
  systemctl status --no-pager "$SERVICE_NAME" || true
}

main() {
  require_root

  if [[ ! -f "$SERVICE_TEMPLATE" ]]; then
    echo "error: missing template $SERVICE_TEMPLATE" >&2
    exit 1
  fi
  if [[ ! -f "$CONFIG_TEMPLATE" ]]; then
    echo "error: missing template $CONFIG_TEMPLATE" >&2
    exit 1
  fi

  build_server
  install_config
  install_service
  restart_service

  echo "deploy finished"
  echo "service : $SERVICE_NAME"
  echo "user    : $RUN_USER"
  echo "workdir : $WORK_DIR"
  echo "music   : $MUSIC_ROOT"
  echo "port    : $PORT"
  echo "config  : $WORK_DIR/music-server.yaml"
}

main "$@"
