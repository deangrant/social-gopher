# Validate site

Run and interpret a scoped catalog self-test.

## Input

- **Required:** site `Name` exactly as in `data/sites.json` (or say "ask me").
- **Optional:** custom `-catalog` path (default: embedded after rebuild).

## Steps

1. If the binary may be stale after catalog edits:

   ```bash
   go build -o social-gopher ./cmd/social-gopher
   ```

2. Run:

   ```bash
   ./social-gopher -validate-catalog -site '<Name>'
   ```

3. Interpret output:
   - `PASS` — claimed found and unclaimed not found.
   - `FAIL` — mismatch; include HTTP status / existence from the log.
   - `SKIP` — missing `username_claimed` / `username_unclaimed`.

4. Do not run unbounded `-validate-catalog` or `-profile full` unless the user
   explicitly requests a broad live run.
