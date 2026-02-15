#!/usr/bin/env bash
set -euo pipefail
shopt -s failglob

TARGET_BIN_PATH="$1"

URL="https://github.com/prometheus/alertmanager/releases/download/v0.31.0/alertmanager-0.31.0.linux-amd64.tar.gz"

wget "${URL}"
tar -xzf "alertmanager-0.31.0.linux-amd64.tar.gz"

mv "alertmanager-0.31.0.linux-amd64/alertmanager" "${TARGET_BIN_PATH}"
