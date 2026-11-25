#!/bin/sh
# Сборка под Windows из Linux/macOS (кросс-компиляция, cgo не нужен)
# и веб-версии (WebAssembly) для показа в браузере.
set -e
cd "$(dirname "$0")/.."
go test ./internal/...
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H=windowsgui" -o dist/Pyatimirye.exe ./cmd/pyatimirye
mkdir -p dist/web
GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w" -o dist/web/game.wasm ./cmd/pyatimirye
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/web/
python3 scripts/make-html.py dist/web dist/Pyatimirye.html
echo "Готово: dist/Pyatimirye.exe, dist/Pyatimirye.html (открыть двойным щелчком) и dist/web/"
