---
name: local-verify
description: >-
  Run the local CI-parity checks for social-gopher before claiming work done or
  committing. Use when finishing a change, before a PR, or when the user asks to
  verify, lint, or test.
---

# Local verify

Run from the repository root. Prefer `/usr/local/go/bin/go` if `go` is not on
`PATH`. Fail the task if any step fails; fix issues and re-run.

## Required

```bash
golangci-lint fmt ./...
golangci-lint run ./...
go test ./... -race -count=1
```

If `golangci-lint` is missing, fall back to:

```bash
gofmt -l .
go test ./... -race -count=1
```

and note that full lint was skipped.

## When catalog or scan behavior changed

```bash
go build -o social-gopher ./cmd/social-gopher
# optional scoped live check:
./social-gopher -validate-catalog -site '<Name>'
```

Avoid unbounded `-validate-catalog` or `-profile full` unless the user explicitly
asks for a broad live run.

## Done criteria

- Format and lint clean (or documented fallback).
- All packages pass under `-race`.
- Catalog embed rebuilt if `data/sites.json` changed.
