#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

# Simple lint: require greet() in src/app.py and no trailing whitespace.
file="$root/src/app.py"
grep -q 'def greet' "$file" || exit 1
if grep -q '[[:space:]]$' "$file"; then
  exit 1
fi
echo "lint clean"
