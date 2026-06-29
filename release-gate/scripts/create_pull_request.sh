#!/usr/bin/env bash
set -euo pipefail

# Side-effect stub: writes a release manifest (no real PR created).
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

printf '{"action":"create_pull_request","title":"Release candidate","status":"would_open"}\n'
