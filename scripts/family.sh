#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# Every repository in the scout family verifies its own row against the
# manifest scout publishes, so the map and the territory cannot drift. This
# checks the facts the row states about this repository — its licence,
# language and lockstep — against what the working tree actually is.
#
#   scripts/family.sh
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

manifest="${SCOUT_ECOSYSTEM_URL:-https://raw.githubusercontent.com/sebastienrousseau/scout/main/ecosystem.json}"
# Fetched to a file and parsed from it, never piped into an interpreter:
# that shape reads as download-then-run to a supply-chain scanner.
mf=$(mktemp)
trap 'rm -f "$mf"' EXIT
curl -fsSL "$manifest" -o "$mf"
row=$(python3 - "$mf" <<'PYEOF'
import json, sys
m = json.load(open(sys.argv[1]))
rows = [r for r in m["repositories"] if r["name"] == "scout-mcp"]
if not rows:
    sys.exit("family: scout-mcp has no row in the family manifest")
r = rows[0]
print(r["status"], r["license"], r["language"], str(r["lockstep"]).lower())
PYEOF
)
read -r status licence language lockstep <<<"$row"

fail=0
# LICENSES/ holds one file per licence in use, named by SPDX identifier.
have=$(basename -s .txt LICENSES/*.txt | head -1)
if [ "$licence" != "$have" ]; then
  echo "family: the manifest says $licence and LICENSES/ holds $have" >&2; fail=1
fi
if [ "$language" != "go" ] || [ ! -f go.mod ]; then
  echo "family: the manifest says $language and this is a Go module" >&2; fail=1
fi
if [ "$lockstep" != "true" ] || [ ! -x scripts/lockstep.sh ]; then
  echo "family: the manifest says lockstep=$lockstep and this repository carries scout's version" >&2; fail=1
fi
if [ "$status" != "shipping" ]; then
  # The row flips to shipping in the scout release after this repository's
  # first; until then the facts above are what can be checked.
  echo "::warning::family: the manifest still lists scout-mcp as $status"
fi
[ "$fail" -eq 0 ] && echo "family: the manifest's row for scout-mcp is true of this tree ($licence, $language, lockstep=$lockstep, $status)"
exit "$fail"
