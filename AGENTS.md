# Agent and contributor guidance

Structured conventions for AI agents and humans working in this repository. For
fuller context, see [README.md](README.md).

## Rules

- [`.agents/rules/`](.agents/rules/) (symlinked from [`.cursor/rules`](.cursor/rules))
- [`.agents/rules/go-project.mdc`](.agents/rules/go-project.mdc) — always-on package map, formatting, Go boundaries
- [`.agents/rules/catalog-schema.mdc`](.agents/rules/catalog-schema.mdc) — catalog JSON schema, profiles, load trust boundary
- [`.agents/rules/security-osint.mdc`](.agents/rules/security-osint.mdc) — always-on OSINT ethics and security invariants

## Skills

- [`.agents/skills/`](.agents/skills/)
- [`.agents/skills/add-catalog-site/`](.agents/skills/add-catalog-site/) — add or edit sites in `data/sites.json`
- [`.agents/skills/local-verify/`](.agents/skills/local-verify/) — lint, race tests, scoped catalog validate
- [`.agents/skills/probe-classifier/`](.agents/skills/probe-classifier/) — status/body/redirect classification invariants
- [`.agents/skills/google-go-style-guide/`](.agents/skills/google-go-style-guide/) — Google Go style
- [`.agents/skills/solid-go-design/`](.agents/skills/solid-go-design/) — SOLID design in Go

## Commands

- [`.agents/commands/`](.agents/commands/) (symlinked from [`.cursor/commands`](.cursor/commands))
- `/verify` — local CI checklist (fmt, lint, race tests)
- `/add-site` — guided catalog site addition
- `/validate-site` — scoped `-validate-catalog -site`

## Hooks

- Config: [`.cursor/hooks.json`](.cursor/hooks.json)
- `afterFileEdit` → [`.agents/hooks/gofmt.sh`](.agents/hooks/gofmt.sh) formats edited `*.go` files
- `beforeShellExecution` → [`.agents/hooks/guard-live-scan.sh`](.agents/hooks/guard-live-scan.sh) asks before `-profile full` or unscoped `-validate-catalog`
