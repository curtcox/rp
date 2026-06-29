#!/usr/bin/env bash
set -euo pipefail

# Deterministic CSV normalizer: trim headers, lowercase names, stable column order.
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

exec python3 - "$root/data/raw.csv" "$root/schema/spec.yaml" <<'PY'
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

raw_path, schema_path = sys.argv[1], sys.argv[2]
spec = load_spec(schema_path)
cols = [c["name"] for c in spec["columns"]]
normalize = {c["name"]: c.get("normalize") for c in spec["columns"]}

out = csv.writer(sys.stdout, lineterminator="\n")
out.writerow(cols)

with open(raw_path, newline="") as f:
    reader = csv.DictReader(f)
    for row in reader:
        cleaned = {}
        for key, val in row.items():
            name = key.strip().lower()
            val = val.strip()
            if normalize.get(name) == "lowercase":
                val = val.lower()
            cleaned[name] = val
        out.writerow([cleaned[c] for c in cols])
PY
