#!/usr/bin/env bash
set -euo pipefail

# Simulates an LLM translation: deterministic French gloss, no network.
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

src="${1:?usage: translate.sh SOURCE.md}"

exec python3 - "$src" <<'PY'
import sys

lines = open(sys.argv[1]).read().splitlines()
out = [
    "# Projet Exemple",
    "",
    "Ce document décrit un petit outil en ligne de commande pour planifier des ressources.",
    "",
    "L'outil lit une configuration YAML et produit des plans fondés sur des preuves.",
    "",
]
sys.stdout.write("\n".join(out))
PY
