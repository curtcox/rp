#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

if command -v pytest >/dev/null 2>&1; then
  exec pytest
fi

python3 - <<'PY'
from greet import greet

assert greet() == "Hello, rp!", greet()
print("ok")
PY
