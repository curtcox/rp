#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

python3 - <<'PY'
import importlib.util
spec = importlib.util.spec_from_file_location("app", "src/app.py")
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)
assert mod.greet() == "hello", mod.greet()
print("tests pass")
PY
