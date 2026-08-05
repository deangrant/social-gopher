#!/usr/bin/env bash
# beforeShellExecution: ask before broad live catalog/scan egress (fail-open).
set -eu

input=$(cat || true)
cmd=""
if command -v node >/dev/null 2>&1; then
  cmd=$(printf '%s' "$input" | node -e '
    let raw = "";
    process.stdin.on("data", (c) => (raw += c));
    process.stdin.on("end", () => {
      try {
        const c = JSON.parse(raw).command;
        if (typeof c === "string") process.stdout.write(c);
      } catch {}
    });
  ' || true)
elif command -v python3 >/dev/null 2>&1; then
  cmd=$(printf '%s' "$input" | python3 -c '
import json, sys
try:
    c = json.load(sys.stdin).get("command") or ""
    if isinstance(c, str):
        sys.stdout.write(c)
except Exception:
    pass
' || true)
fi

if [ -z "${cmd:-}" ]; then
  printf '%s\n' '{ "permission": "allow" }'
  exit 0
fi

ask=false
reason=""

# Match CLI invocations, not incidental path segments like .../social-gopher/.cursor/...
if printf '%s' "$cmd" | grep -Eq -- '(^|[[:space:]=./])social-gopher([[:space:]]|$)' \
  || printf '%s' "$cmd" | grep -Eq -- 'go[[:space:]]+run[[:space:]].*cmd/social-gopher'; then
  if printf '%s' "$cmd" | grep -Eq -- '(^|[[:space:]])(-profile|--profile)[[:space:]]+full([[:space:]]|$)'; then
    ask=true
    reason="Command uses -profile full (broad live egress)."
  fi
  if printf '%s' "$cmd" | grep -Eq -- '(^|[[:space:]])(-validate-catalog|--validate-catalog)([[:space:]]|$)'; then
    if ! printf '%s' "$cmd" | grep -Eq -- '(^|[[:space:]])(-site|--site)[[:space:]]+'; then
      ask=true
      if [ -n "$reason" ]; then
        reason="$reason Unscoped -validate-catalog."
      else
        reason="Command runs -validate-catalog without -site (broad live egress)."
      fi
    fi
  fi
fi

if [ "$ask" = true ]; then
  if command -v python3 >/dev/null 2>&1; then
    reason_json=$(printf '%s' "$reason" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')
  else
    reason_json='"Review broad live scan command."'
  fi
  printf '{"permission":"ask","user_message":%s,"agent_message":%s}\n' \
    "$reason_json" "$reason_json"
  exit 0
fi

printf '%s\n' '{ "permission": "allow" }'
exit 0
