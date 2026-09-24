#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# The version rule: every repository in the scout family carries scout's
# version. This one wraps scout's release — its container image is built on
# scout's image and its CI evaluates itself with scout at that tag — so its
# version is scout's latest release, exactly.
#
#   scripts/lockstep.sh
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

mine=$(grep -Eo '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | tr -d '#[] ')
[ -n "$mine" ] || { echo "lockstep: CHANGELOG.md has no released version heading" >&2; exit 1; }

api="https://api.github.com/repos/sebastienrousseau/scout/releases/latest"
auth=()
[ -n "${GITHUB_TOKEN:-}" ] && auth=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
# Fetched to a file and parsed from it, never piped into an interpreter:
# that shape reads as download-then-run to a supply-chain scanner.
release=$(mktemp)
trap 'rm -f "$release"' EXIT
curl -fsSL "${auth[@]}" -H "Accept: application/vnd.github+json" "$api" -o "$release"
theirs=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["tag_name"].lstrip("v"))' "$release")

if [ "$mine" = "$theirs" ]; then
  echo "lockstep: $mine matches scout's latest release"
else
  echo "lockstep: this repository is at $mine and scout's latest release is $theirs; release scout-mcp at $theirs, on scout's image for that release" >&2
  exit 1
fi
