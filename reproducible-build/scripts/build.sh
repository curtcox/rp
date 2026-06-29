#!/usr/bin/env bash
set -euo pipefail

# Tutorial script: a deterministic "build". It bundles every file under src/
# into a single artifact written to stdout. The output depends only on the
# source bytes — no timestamps, no host paths, files emitted in a stable
# (LC_ALL=C) order — so identical inputs always yield identical output. That is
# what makes the build *reproducible*, and what lets rp record a hash that two
# independent builds can be compared against.
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

printf '# rp reproducible-build artifact v1\n'
find src -type f | LC_ALL=C sort | while read -r f; do
  printf '## %s\n' "$f"
  cat "$f"
done
