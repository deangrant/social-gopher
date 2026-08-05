#!/usr/bin/env bash
# afterFileEdit: format edited Go files with gofmt (fail-open).
set -eu

input=$(cat || true)
file=""
if command -v node >/dev/null 2>&1; then
  file=$(printf '%s' "$input" | node -e '
    let raw = "";
    process.stdin.on("data", (c) => (raw += c));
    process.stdin.on("end", () => {
      try {
        const p = JSON.parse(raw).file_path;
        if (typeof p === "string" && p) process.stdout.write(p);
      } catch {}
    });
  ' || true)
elif command -v python3 >/dev/null 2>&1; then
  file=$(printf '%s' "$input" | python3 -c '
import json, sys
try:
    p = json.load(sys.stdin).get("file_path") or ""
    if isinstance(p, str):
        sys.stdout.write(p)
except Exception:
    pass
' || true)
fi

[ -z "${file:-}" ] && exit 0
[ ! -f "$file" ] && exit 0
case "$file" in
  *.go) ;;
  *) exit 0 ;;
esac

gofmt_bin=""
if command -v gofmt >/dev/null 2>&1; then
  gofmt_bin=$(command -v gofmt)
elif [ -x /usr/local/go/bin/gofmt ]; then
  gofmt_bin=/usr/local/go/bin/gofmt
fi

if [ -n "$gofmt_bin" ]; then
  "$gofmt_bin" -w "$file" || true
fi
exit 0
