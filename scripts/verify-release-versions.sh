#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# Fail unless every place that names the version being released names it:
# the CHANGELOG heading, the README's install snippets, the registry
# listing in server.json, and the scout release the Dockerfile builds on.
#
#   scripts/verify-release-versions.sh v0.0.5
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
tag="${1:-${GITHUB_REF_NAME:-}}"
[ -n "$tag" ] || { echo "usage: $0 vX.Y.Z" >&2; exit 2; }
ver="${tag#v}"
fail=0

grep -Eq "^## \[$ver\]" CHANGELOG.md || { echo "CHANGELOG.md has no '## [$ver]' heading" >&2; fail=1; }

# Both install lines: scout-mcp's, and the scout it runs.
if grep -Eo 'sebastienrousseau/scout(-mcp)?/cmd/scout(-mcp)?@v[0-9]+\.[0-9]+\.[0-9]+' README.md | grep -v "@v$ver\$"; then
  echo "README.md pins a go install version other than $ver" >&2; fail=1
fi
if grep -Eo 'ghcr\.io/sebastienrousseau/scout-mcp:[0-9]+\.[0-9]+\.[0-9]+' README.md docs/*.md | grep -v ":$ver\$"; then
  echo "the docs pin an image version other than $ver" >&2; fail=1
fi

# server.json: the listing's version and the image it points at.
listed=$(python3 - "$ver" <<'PYEOF'
import json, sys
ver = sys.argv[1]
s = json.load(open("server.json"))
bad = []
if s.get("version") != ver:
    bad.append("version is %r" % s.get("version"))
for p in s.get("packages", []):
    if p.get("registryType") == "oci" and not p["identifier"].endswith(":" + ver):
        bad.append("OCI identifier is %r" % p["identifier"])
print("; ".join(bad))
PYEOF
)
if [ -n "$listed" ]; then
  echo "server.json disagrees with $ver: $listed" >&2; fail=1
fi

# The image is built on scout's image for the same release.
base=$(sed -nE 's/^ARG SCOUT_VERSION=([0-9.]+)$/\1/p' Dockerfile)
if [ "$base" != "$ver" ]; then
  echo "Dockerfile builds on scout ${base:-<none>}, not $ver; move the FROM digest and SCOUT_VERSION together" >&2; fail=1
fi

[ "$fail" -eq 0 ] && echo "release versions agree on $ver"
exit "$fail"
