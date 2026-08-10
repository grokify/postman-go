#!/bin/bash
# generate.sh - Generate the internal Go API client from the OpenAPI spec using ogen.
#
# Unlike providers that publish a downloadable OpenAPI document, Postman does not.
# The spec at openapi/openapi.yaml is hand-authored and maintained in-repo (see the
# header comment in that file). This script only runs the generator over it.
#
# Usage:
#   ./generate.sh
#
# Prerequisites: Go toolchain (ogen is invoked via `go run`, so no install needed).

set -euo pipefail

OGEN_VERSION="v1.24.0"
SPEC="openapi/openapi.yaml"
TARGET="internal/api"

if [ ! -f "$SPEC" ]; then
  echo "Error: $SPEC not found." >&2
  exit 1
fi

echo "Generating API client with ogen $OGEN_VERSION..."
go run "github.com/ogen-go/ogen/cmd/ogen@${OGEN_VERSION}" \
  --package api \
  --target "$TARGET" \
  --clean \
  "$SPEC"

echo ""
echo "Running go mod tidy..."
go mod tidy

echo ""
echo "Verifying build..."
go build ./...

echo ""
echo "Done. Generated client is in $TARGET/."
