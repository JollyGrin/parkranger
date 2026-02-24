#!/usr/bin/env bash
set -euo pipefail
echo "Building parkranger..."
go build -o parkranger ./cmd/parkranger
echo "Built ./parkranger"
