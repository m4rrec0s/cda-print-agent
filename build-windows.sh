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
VERSION="${1:-$(node -p 'require("./wails.json").info.productVersion')}"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Versao invalida: $VERSION (use MAJOR.MINOR.PATCH)" >&2
  exit 1
fi

echo "==> Compilando frontend..."
cd frontend
npm install --silent
npm run build
cd ..

echo "==> Buildando para Windows amd64..."
wails build -platform windows/amd64 -o CdAPrintAgent.exe -clean -ldflags "-X main.Version=$VERSION"

OUTPUT_DIR="$SCRIPT_DIR/build/bin"
echo ""
echo "✅ Build concluido!"
echo "📦 Executavel: $OUTPUT_DIR/CdAPrintAgent.exe"
