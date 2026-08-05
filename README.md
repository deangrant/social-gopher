# Social Gopher

Social Gopher is a command-line OSINT tool that checks whether a username exists on many social networks and websites. It probes sites concurrently and prints matches with profile URLs. Optional CSV export is available via `-csv`.

## Install

```bash
go install github.com/deangrant/social-gopher/cmd/social-gopher@latest
```

Or from a clone:

```bash
go build -o social-gopher ./cmd/social-gopher
```

Requires Go 1.22+.

## Usage

```bash
social-gopher [flags] <username>
```

Examples:

```bash
# Scan the default profile (initial seed only)
social-gopher alice

# default + one profile, or combine several (non-cumulative)
social-gopher -profile developer alice
social-gopher -profile developer -profile community alice
social-gopher -profile full alice

# Verbose + CSV export
social-gopher -v -csv alice

# Limit to specific sites
social-gopher -site GitHub -site GitLab alice

# Route through local Tor
social-gopher -tor alice

# Custom SOCKS/HTTP proxy
social-gopher -proxy socks5h://127.0.0.1:9050 alice
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-csv` | off | Write found results to a sanitized `{username}.csv` in the current directory (path separators replaced) |
| `-tor` | off | Use Tor (reuse existing SOCKS or start system `tor`) |
| `-proxy` | | Proxy URL (`socks5h://`, `socks5://`, `http://`, `https://`); SOCKS schemes resolve DNS through the proxy |
| `-timeout` | `20s` | Per-site probe budget (shared by HEAD and any GET retry) |
| `-workers` | `20` | Concurrent workers (must be ≥ 1) |
| `-profile` | `default` | Repeatable scan profile: `default`, `developer`, `creative`, `community`, or `full` |
| `-site` | all | Repeatable site name filter |
| `-nsfw` | off | Include NSFW-tagged sites |
| `-v`, `-verbose` | off | Print non-hits and errors |
| `-catalog` | bundled | Path to a custom `sites.json` (trust boundary: only load files you trust) |
| `-validate-catalog` | off | Probe `username_claimed` / `username_unclaimed` self-tests (no scan username) |

`-profile` selects site groups as `default` plus each named profile (non-cumulative; repeatable). `full` scans the entire catalog. Applied before `-site` / `-nsfw`.

If both `-tor` and `-proxy` are set, `-proxy` wins and a notice is printed.

## Tor

1. Install Tor so the `tor` binary is on your `PATH` (e.g. `apt install tor`, Homebrew).
2. Run with `-tor`. If a SOCKS5 proxy is already responding on `127.0.0.1:9050`, that listener is reused; otherwise Social Gopher starts a temporary Tor process and stops it when the scan finishes. A non-SOCKS process on that port is an error.
3. Or point at any proxy with `-proxy socks5h://127.0.0.1:9050` (skips process management).

First bootstrap of a freshly started Tor can take ~30–60s. Both `socks5h` and `socks5` send DNS through the proxy (curl’s `socks5h` semantics; unlike curl’s local-DNS `socks5`). Prefer `socks5h` in URLs. Tor exits are often rate-limited or blocked by WAFs, so coverage may drop compared to a direct scan.

## Site catalog

The default catalog is embedded from [`data/sites.json`](data/sites.json). It is an original curated list (400+ sites across developer, creative, music, gaming, finance, regional, and niche communities) using status / body / redirect checks. Add sites by editing that file; no code changes are required. Grow coverage in verified batches using `-validate-catalog`.

Custom `-catalog` files are a trust boundary. Load rejects non-`http`/`https` URLs, private/link-local/metadata hosts, `POST`, dangerous hop-by-hop/`Host` headers, and invalid or over-long `username_pattern` values. DNS rebinding is a residual risk if a hostname later resolves to a private address.

### Schema

```json
{
  "sites": [
    {
      "name": "GitHub",
      "home_url": "https://github.com",
      "profile_url": "https://github.com/{username}",
      "probe_url": "",
      "method": "GET",
      "headers": {},
      "check": {
        "type": "status",
        "not_found_status": [404],
        "not_found_text": ["user not found"]
      },
      "username_pattern": "^[a-zA-Z0-9-]+$",
      "username_claimed": "torvalds",
      "username_unclaimed": "zzsgopher9x8q7w6e5r4t3y",
      "profile": "default",
      "nsfw": false
    }
  ]
}
```

`profile` is required and is the introducing wave for the site: `default`, `developer`, `creative`, or `community`.

| `check.type` | Meaning |
|--------------|---------|
| `status` | Does not follow redirects; exists on first-response 2xx unless status is in `not_found_status` (default `[404]`) or optional `not_found_text` matches the body (soft-404; forces GET); other codes (including 3xx) are unknown |
| `body` | Does not follow redirects; missing if the first-response body contains any `not_found_text` substring; else exists on 2xx |
| `redirect` | Does not follow redirects; exists on 2xx, otherwise missing |

Use `{username}` in `profile_url` / `probe_url`. `probe_url` is optional when the check URL differs from the public profile link.

`method`, when set, must be `GET` or `HEAD` (default depends on check type).

`not_found_text` is required for `body` checks and optional on `status` as a soft-404 guard.

`username_claimed` / `username_unclaimed` are optional self-test fixtures (ignored during normal scans).

### Verifying a new site

1. Add the site row with a working `check` type.
2. Set `username_claimed` (known existing account) and `username_unclaimed` (guaranteed missing).
3. Validate:

```bash
social-gopher -validate-catalog -site YourSite
```

4. Expect `PASS YourSite`. Sites missing self-test fields are reported as `SKIP`.
5. Rebuild so the embedded catalog picks up changes: `go build -o social-gopher ./cmd/social-gopher`

To validate every site that has fixtures:

```bash
social-gopher -validate-catalog
```

## Output

- **Terminal** (default): white live checking status with a rotating `[|] [/] [-] [\]` spinner (TTY only; cleared when done), green `[+] Site:` for found profiles, then a summary. Non-hits stay off-screen unless `-v`.
- **CSV** (`-csv`): `username,name,home_url,profile_url,exists,http_status,response_time_s` (found rows only). The file is written as a sanitized `{username}.csv` in the current working directory (`/`, `\`, and NUL become `_`). Cells starting with `=`, `+`, `-`, `@`, tab, or CR are prefixed with `'` to avoid spreadsheet formula injection.

Colors and the ephemeral status line are enabled only when stdout is a terminal.

## Project layout

```
cmd/social-gopher/     CLI composition root
internal/catalog/      Site catalog load + filter
internal/scan/         Concurrent probes + classifiers
internal/report/       Terminal + CSV writers
internal/transport/    HTTP client, Tor/proxy
internal/torrun/       Optional system Tor process lifecycle
data/sites.json        Curated catalog (embedded)
```

## Lint

```bash
# install: https://golangci-lint.run/docs/welcome/install/
golangci-lint fmt ./...
golangci-lint run ./...
```

## Ethics

Use only for lawful purposes and with permission where required. Respect site terms of service and local law. This tool is for defensive OSINT and research, not harassment or unauthorized access.

## License

MIT. See [LICENSE](LICENSE).
