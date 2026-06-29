#!/usr/bin/env bash
set -euo pipefail

# Independent verifier: checks the translation matches the expected deterministic output.
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

doc="${1:?usage: check_translation.sh TRANSLATED.md}"

exec python3 - "$doc" <<'PY'
import hashlib
import sys

text = open(sys.argv[1]).read()
expected_lines = [
    "# Projet Exemple",
    "",
    "Ce document décrit un petit outil en ligne de commande pour planifier des ressources.",
    "",
    "L'outil lit une configuration YAML et produit des plans fondés sur des preuves.",
    "",
]
expected = "\n".join(expected_lines)
if text != expected:
    sys.stderr.write("translation mismatch\n")
    sys.exit(1)
digest = hashlib.sha256(text.encode()).hexdigest()
print(f"translation verified sha256={digest}")
PY
