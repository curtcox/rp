#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

# Toy secret scanner: fail if TODO(hack) marker present.
if grep -R 'TODO(hack)' src/ 2>/dev/null; then
  exit 1
fi
echo "scan clean"
