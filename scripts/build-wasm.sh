#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/dist/wasm"
GOROOT="$(go env GOROOT)"

cd "$ROOT"
mkdir -p "$OUT"
GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath -o "$OUT/jed-core.wasm" ./cmd/jed-wasm
cp "$GOROOT/misc/wasm/wasm_exec.js" "$OUT/wasm_exec.js"
cp "$ROOT/web/wasm/jed-core.js" "$OUT/jed-core.js"
printf 'WASM criado em %s\n' "$OUT"
