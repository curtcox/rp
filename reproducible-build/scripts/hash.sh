#!/usr/bin/env bash
set -euo pipefail

# Print the SHA-256 of a file, using whichever tool the host provides.
# Reads stdin when no argument is given.
target="${1:--}"

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$target" | awk '{print $1}'
else
  shasum -a 256 "$target" | awk '{print $1}'
fi
