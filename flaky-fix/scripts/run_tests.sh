#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

port="$(grep '^PORT=' config/app.env | cut -d= -f2)"
if [ "$port" != "8080" ]; then
  echo "service misconfigured: PORT=$port (expected 8080)" >&2
  exit 1
fi
echo "tests pass"
