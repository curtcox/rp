#!/usr/bin/env bash
set -euo pipefail

# Tutorial script: build the project twice, independently, and compare the
# artifact hashes. Exit 0 only when both builds are byte-for-byte identical.
# rp turns that exit code into a `build_reproducible` assertion at `reproduced`
# confidence — evidence that the system reproduced the result itself, rather
# than merely observing a single build succeed.
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

a="$(./scripts/build.sh | ./scripts/hash.sh)"
b="$(./scripts/build.sh | ./scripts/hash.sh)"

if [ "$a" = "$b" ]; then
  printf 'reproducible: %s\n' "$a"
  exit 0
fi

printf 'NOT reproducible: %s != %s\n' "$a" "$b" >&2
exit 1
