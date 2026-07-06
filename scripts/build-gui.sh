#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GUI_DIR="$ROOT/gui"
DIST_DIR="$ROOT/internal/gui/dist"

cd "$GUI_DIR"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"
cp -a "$GUI_DIR/build/." "$DIST_DIR/"

echo "GUI assets copied to $DIST_DIR"
