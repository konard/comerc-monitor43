#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG="$SCRIPT_DIR/../.golangci.yml"

MODULE_NAME="$(awk '/^module/{print $2; exit}' go.mod)"

TMP_BASE=$(mktemp -t golangci)
TMP_CONFIG="${TMP_BASE}.yml"
mv "$TMP_BASE" "$TMP_CONFIG"
trap 'rm -f "$TMP_CONFIG"' EXIT

sed "s|{{MODULE_NAME}}|$MODULE_NAME|g" "$CONFIG" > "$TMP_CONFIG"

golangci-lint run --config "$TMP_CONFIG" ./...
