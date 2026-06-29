#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

python3 - <<'PY'
from pathlib import Path

path = Path("config/app.env")
text = path.read_text()
if "PORT=9999" in text:
    path.write_text(text.replace("PORT=9999", "PORT=8080"))
elif "PORT=8080" not in text:
    raise SystemExit("unexpected config state")
print("config repaired")
PY
