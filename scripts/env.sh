#!/usr/bin/env bash
# Source this file in a Linux cloud task to select the pinned toolchain.
export ASTERIADE_GO_VERSION=1.27.2
export ASTERIADE_TOOL_ROOT="${ASTERIADE_TOOL_ROOT:-/workspace/.tools}"
export GOPATH="${GOPATH:-/workspace/.cache/gopath}"
export GOCACHE="${GOCACHE:-/workspace/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-/workspace/.cache/gomod}"
export GOTOOLCHAIN=local
export PATH="$ASTERIADE_TOOL_ROOT/go-$ASTERIADE_GO_VERSION/go/bin:$PATH"
