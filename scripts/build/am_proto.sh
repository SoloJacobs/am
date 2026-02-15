#!/bin/bash
set -euo pipefail
shopt -s failglob

TARGET_BIN_PATH="$1"

GO_VERSION="1.25.6"
REPO_PATH="/am"

git clone --no-tags --depth 1 --branch main --single-branch https://github.com/prometheus/alertmanager .

docker run --rm \
  -v "$(pwd):$REPO_PATH" \
  -w "$REPO_PATH" \
  -e CGO_ENABLED=0 \
  golang:$GO_VERSION \
  sh -c "go build -trimpath -buildvcs=false -ldflags '-s -w' -o bin/am ./cmd/alertmanager && cat bin/am" >"$TARGET_BIN_PATH"

chmod +x "$TARGET_BIN_PATH"
