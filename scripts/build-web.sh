#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Vercel's Next.js build image may not expose the Go compiler. Reuse Go when
# available (local/CI); otherwise install the exact Go release used by RC1.8.
if ! command -v go >/dev/null 2>&1; then
    GO_VERSION="1.23.12"

    case "$(uname -m)" in
        x86_64|amd64)
            GO_ARCH="amd64"
            GO_SHA256="d3847fef834e9db11bf64e3fb34db9c04db14e068eeb064f49af747010454f90"
            ;;
        aarch64|arm64)
            GO_ARCH="arm64"
            GO_SHA256="52ce172f96e21da53b1ae9079808560d49b02ac86cecfa457217597f9bc28ab3"
            ;;
        *)
            echo "Arquitetura não suportada para instalação automática do Go: $(uname -m)"
            exit 1
            ;;
    esac

    GO_ARCHIVE="/tmp/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    GO_INSTALL_DIR="/tmp/jed-go"

    echo "Go não encontrado. Instalando Go ${GO_VERSION} para o build..."
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o "$GO_ARCHIVE"
    echo "${GO_SHA256}  ${GO_ARCHIVE}" | sha256sum -c -

    rm -rf "$GO_INSTALL_DIR"
    mkdir -p "$GO_INSTALL_DIR"
    tar -C "$GO_INSTALL_DIR" -xzf "$GO_ARCHIVE"

    export GOROOT="$GO_INSTALL_DIR/go"
    export PATH="$GOROOT/bin:$PATH"
fi

echo "Usando $(go version)"

cd "$ROOT"
bash scripts/build-wasm.sh

mkdir -p "$ROOT/web/public/wasm"
cp "$ROOT/dist/wasm/jed-core.wasm" "$ROOT/web/public/wasm/jed-core.wasm"
cp "$ROOT/dist/wasm/wasm_exec.js" "$ROOT/web/public/wasm/wasm_exec.js"
cp "$ROOT/dist/wasm/jed-core.js" "$ROOT/web/public/wasm/jed-core.js"

mkdir -p "$ROOT/web/public/data"
cp "$ROOT/catalogo_negocios.json" "$ROOT/web/public/data/catalogo_negocios.json"
cp "$ROOT/catalogo_insumos.json" "$ROOT/web/public/data/catalogo_insumos.json"

npm --prefix "$ROOT/web" run build
