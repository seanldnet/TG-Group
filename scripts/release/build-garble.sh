#!/usr/bin/env bash
set -Eeuo pipefail

: "${GOOS:?GOOS is required}"
: "${GOARCH:?GOARCH is required}"
: "${VERSION:?VERSION is required}"
: "${COMMIT:?COMMIT is required}"
: "${BUILD_TIME:?BUILD_TIME is required}"

GARBLE_VERSION="v0.14.2"
GARBLE_ACTUAL="$(garble version 2>&1)"
grep -Fq "$GARBLE_VERSION" <<< "$GARBLE_ACTUAL" || {
  printf 'Expected Garble %s, got: %s\n' "$GARBLE_VERSION" "$GARBLE_ACTUAL" >&2
  exit 1
}

mkdir -p dist
output="dist/tg-group-${GOOS}-${GOARCH}"
(
  cd backend
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    garble -literals -tiny build -trimpath \
      -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.builtAt=$BUILD_TIME" \
      -o "../$output" ./cmd/server
)

chmod 0755 "$output"
garble version > "dist/GARBLE-VERSION-$GOOS-$GOARCH.txt"
{
  printf 'version=%s\n' "$VERSION"
  printf 'commit=%s\n' "$COMMIT"
  printf 'built_at=%s\n' "$BUILD_TIME"
  printf 'go_version=%s\n' "$(go version)"
  printf 'garble_version=%s\n' "$GARBLE_VERSION"
  printf 'goos=%s\n' "$GOOS"
  printf 'goarch=%s\n' "$GOARCH"
  printf 'garble_flags=-literals -tiny\n'
  printf 'build_flags=-trimpath -ldflags=-s-w\n'
} > "dist/BUILD-METADATA-$GOOS-$GOARCH.txt"
(
  cd dist
  sha256sum "$(basename "$output")" > "$(basename "$output").sha256"
)
