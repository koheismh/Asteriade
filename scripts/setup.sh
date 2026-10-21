#!/usr/bin/env bash
set -euo pipefail
ASTERIADE_REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
source "$ASTERIADE_REPO_ROOT/scripts/env.sh"
if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]]; then
  printf '%s\n' 'This bootstrap supports Linux x86_64. On Windows install the pinned Go release normally.' >&2
  exit 1
fi
ASTERIADE_ARCHIVE="$ASTERIADE_TOOL_ROOT/go$ASTERIADE_GO_VERSION.linux-amd64.tar.gz"
ASTERIADE_CHECKSUM=ecbadb99091a3f46e31f5f934b068b1864eafa7995211b39eaddf76996045fe5
mkdir -p "$ASTERIADE_TOOL_ROOT" "$GOPATH" "$GOCACHE" "$GOMODCACHE"
if [[ ! -f "$ASTERIADE_ARCHIVE" ]]; then
  curl --fail --location --proto '=https' --proto-redir '=https' --retry 2 \
    "https://go.dev/dl/go$ASTERIADE_GO_VERSION.linux-amd64.tar.gz" -o "$ASTERIADE_ARCHIVE"
fi
printf '%s  %s\n' "$ASTERIADE_CHECKSUM" "$ASTERIADE_ARCHIVE" | sha256sum --check --status
if [[ ! -x "$ASTERIADE_TOOL_ROOT/go-$ASTERIADE_GO_VERSION/go/bin/go" ]]; then
  mkdir -p "$ASTERIADE_TOOL_ROOT/go-$ASTERIADE_GO_VERSION"
  tar -xzf "$ASTERIADE_ARCHIVE" -C "$ASTERIADE_TOOL_ROOT/go-$ASTERIADE_GO_VERSION"
fi
go version
cd "$ASTERIADE_REPO_ROOT"
go mod download
go mod verify
printf '%s\n' 'Setup complete. Source scripts/env.sh before Go commands in a new shell.'
