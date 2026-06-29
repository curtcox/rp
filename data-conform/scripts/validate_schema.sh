#!/usr/bin/env bash
set -euo pipefail

# Validates a cleaned CSV against schema/spec.yaml (reads artifact path from argv).
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

csv_path="${1:?usage: validate_schema.sh PATH_TO_CSV}"
schema="$root/schema/spec.yaml"

exec python3 - "$csv_path" "$schema" <<'PY'
import csv
import sys

def load_spec(path):
    spec = {"columns": [], "row_count": 0}
    col = None
    for line in open(path):
        line = line.strip()
        if line.startswith("- name:"):
            col = {"name": line.split(":", 1)[1].strip()}
            spec["columns"].append(col)
        elif col is not None and line.startswith("type:"):
            col["type"] = line.split(":", 1)[1].strip()
        elif col is not None and line.startswith("normalize:"):
            col["normalize"] = line.split(":", 1)[1].strip()
        elif line.startswith("row_count:"):
            spec["row_count"] = int(line.split(":", 1)[1].strip())
    return spec

csv_path, schema_path = sys.argv[1], sys.argv[2]
spec = load_spec(schema_path)
cols = [c["name"] for c in spec["columns"]]
types = {c["name"]: c["type"] for c in spec["columns"]}
normalize = {c["name"]: c.get("normalize") for c in spec["columns"]}
expected_rows = spec.get("row_count", 0)

with open(csv_path, newline="") as f:
    reader = csv.reader(f)
    header = next(reader)
    if header != cols:
        sys.exit(f"header mismatch: {header!r} != {cols!r}")
    rows = list(reader)

if len(rows) != expected_rows:
    sys.exit(f"row count {len(rows)} != expected {expected_rows}")

for i, row in enumerate(rows, start=1):
    if len(row) != len(cols):
        sys.exit(f"row {i} column count mismatch")
    for name, val, typ in zip(cols, row, [types[c] for c in cols]):
        if typ == "integer" and not val.isdigit():
            sys.exit(f"row {i} {name}: not integer")
        if typ == "float":
            try:
                float(val)
            except ValueError:
                sys.exit(f"row {i} {name}: not float")
        if normalize.get(name) == "lowercase" and val != val.lower():
            sys.exit(f"row {i} {name}: not lowercase")

print("schema validation ok")
PY
