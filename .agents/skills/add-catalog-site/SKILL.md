---
name: add-catalog-site
description: >-
  Add or edit a site in data/sites.json with the correct schema, check type,
  profile, and optional self-test usernames. Use when adding catalog sites,
  fixing site JSON, choosing status/body/redirect checks, or preparing
  username_claimed/unclaimed for -validate-catalog.
---

# Add catalog site

## Steps

1. Read `.agents/rules/catalog-schema.mdc` and a nearby entry in `data/sites.json`
   for formatting consistency (indentation, field order).
2. Choose `check.type`:
   - `status` — existence from HTTP status (optional soft-404 `not_found_text`).
   - `body` — existence from body substring(s) in `not_found_text`.
   - `redirect` — 2xx on first response = found (no follow).
3. Set `profile` to the introducing wave: `default`, `developer`, `creative`,
   or `community`. Set `"nsfw": true` only when appropriate.
4. Append a minimal valid site object. Prefer GET (omit `method`). Include
   `{username}` in `profile_url` (and `probe_url` if used).
5. Optional self-test fixtures: real `username_claimed` (exists) and a long
   random `username_unclaimed` (should not exist).
6. Verify:

   ```bash
   go test ./internal/catalog/ -count=1
   go build -o social-gopher ./cmd/social-gopher
   ./social-gopher -validate-catalog -site '<Name>'
   ```

   Expect `PASS <Name>`, or `SKIP` if claimed/unclaimed are omitted.

## Minimal example

```json
{
  "name": "Example",
  "home_url": "https://example.com",
  "profile_url": "https://example.com/{username}",
  "check": { "type": "status", "not_found_status": [404] },
  "profile": "default"
}
```

## Do not

- Add unknown JSON keys.
- Use `javascript:`, `file:`, `ftp:`, loopback, or private hosts.
- Use `POST` or set `Host` / hop-by-hop headers.
