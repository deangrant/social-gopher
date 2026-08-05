---
name: probe-classifier
description: >-
  Explain or change social-gopher probe classification in internal/scan. Use when
  editing Check/classify logic, redirect policy, soft-404 text, body reads, or
  status/body/redirect matrix tests.
---

# Probe classifier

## Invariants (do not change without an explicit product decision)

- Probes never follow redirects (`http.ErrUseLastResponse` via client clone).
- Status/body: first-response `302` (and other non-2xx outside not-found rules)
  is **not** treated as found. Today status maps those to `unknown`.
- Soft-404: `not_found_text` on a status check forces GET and may mark not found
  on 2xx when the body matches.
- Body checks require a successful body read; read errors => `error`.
- Status checks without `not_found_text` must not buffer response bodies.

## classify sketch

| Type | Found | Not found | Else |
|------|-------|-----------|------|
| status | 2xx and not soft-404 | status in `not_found_status` or soft-404 text | unknown |
| body | 2xx and no not_found_text | any not_found_text match | unknown |
| redirect | 2xx | everything else | — |

## When editing

1. Update [`internal/scan/scan.go`](internal/scan/scan.go) minimally.
2. Extend table-driven coverage in `TestClassifyMatrix` / redirect tests.
3. Run:

   ```bash
   go test ./internal/scan/ -race -count=1
   ```

## Doer note

`Scanner` takes `*http.Client` only (no separate Doer). Redirect policy is
enforced by cloning the client with `CheckRedirect` set.
