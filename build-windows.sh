#!/usr/bin/env bash
set -euo pipefail

# ────────────────────────────────────────────
# Build do CdA Print Agent para Windows
# ────────────────────────────────────────────
# Pré-requisitos:
#   - Go 1.23+
#   - Wails CLI v2 (go install github.com/wailsapp/wails/v2/cmd/wails@latest)
#   - MinGW-w64 (sudo apt install gcc-mingw-w64-x86-64)
#   - Node.js 18+
# ────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "==> Compilando frontend..."
cd frontend
npm install --silent
npm run build
cd ..

echo "==> Buildando para Windows amd64..."
wails build -platform windows/amd64 -o CdAPrintAgent.exe -clean

OUTPUT_DIR="$SCRIPT_DIR/build/bin"
echo ""
echo "✅ Build concluido!"
echo "📦 Executavel: $OUTPUT_DIR/CdAPrintAgent.exe"
