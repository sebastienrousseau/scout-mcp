#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# The image scout-mcp builds on must be scout's image for the version the
# Dockerfile names. SCOUT_VERSION says which release; the FROM digest is
# what is actually pulled. This fails when they disagree, so the image can
# never quietly carry a scout other than the one it claims.
#
#   scripts/verify-digest.sh
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

version=$(sed -nE 's/^ARG SCOUT_VERSION=([0-9.]+)$/\1/p' Dockerfile)
pinned=$(sed -nE 's#^FROM ghcr\.io/sebastienrousseau/scout@(sha256:[0-9a-f]{64})$#\1#p' Dockerfile)
[ -n "$version" ] || { echo "verify-digest: the Dockerfile names no SCOUT_VERSION" >&2; exit 1; }
[ -n "$pinned" ] || { echo "verify-digest: the Dockerfile does not build FROM scout's image by digest" >&2; exit 1; }

tokfile=$(mktemp)
trap 'rm -f "$tokfile"' EXIT
curl -fsSL "https://ghcr.io/token?scope=repository:sebastienrousseau/scout:pull" -o "$tokfile"
token=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["token"])' "$tokfile")
published=$(curl -fsSI -H "Authorization: Bearer ${token}" \
  -H "Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json" \
  "https://ghcr.io/v2/sebastienrousseau/scout/manifests/${version}" \
  | tr -d '\r' | awk 'tolower($1) == "docker-content-digest:" { print $2 }')
[ -n "$published" ] || { echo "verify-digest: ghcr.io has no scout image tagged ${version}" >&2; exit 1; }
if [ "$pinned" != "$published" ]; then
  echo "verify-digest: the Dockerfile pins ${pinned} but ghcr.io tags scout ${version} as ${published}" >&2
  exit 1
fi
echo "verify-digest: the Dockerfile builds on scout ${version} (${published})"
