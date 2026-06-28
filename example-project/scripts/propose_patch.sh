#!/usr/bin/env bash
set -euo pipefail

# Tutorial script: emit a unified diff for the bug described in bug.md.
# The bug report path is passed for realism; content is fixed for v0.1.
_="${1:?bug report path required}"

cat <<'PATCH'
diff --git a/greet.py b/greet.py
--- a/greet.py
+++ b/greet.py
@@ -1,2 +1,2 @@
 def greet():
-    return "Hello, World!"
+    return "Hello, rp!"
PATCH
