#!/usr/bin/env bash

set -euo pipefail

BIN_NAME="wigo"
INSTALL_DIR="/usr/local/bin"

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed or not in PATH"
  exit 1
fi

echo "Building ${BIN_NAME}..."
go build -o "${BIN_NAME}" ./cmd/wigo/main.go

echo "Installing to ${INSTALL_DIR}..."
if [ ! -w "${INSTALL_DIR}" ]; then
  sudo mv "${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
else
  mv "${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
fi

echo "Installed ${BIN_NAME} to ${INSTALL_DIR}"
echo "You can now run: ${BIN_NAME}"
