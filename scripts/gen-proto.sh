#!/usr/bin/env bash
set -euo pipefail

protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/common.proto proto/trade.proto proto/recommendation.proto proto/analysis.proto

echo "Protobuf generation complete"


