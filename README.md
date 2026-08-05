# Social Gopher

Social Gopher is a command-line OSINT tool that checks whether a username
exists on many social networks and websites. It probes sites concurrently,
prints matches with profile URLs, and can export found results to CSV. Optional
Tor and proxy routing are supported.

## Install

From a clone:

```bash
go build -o social-gopher ./cmd/social-gopher
```

Requires Go 1.26+ (same minor as [`go.mod`](go.mod)).

## Quick start

```bash
# Default profile (initial seed sites)
social-gopher alice

# default + named profiles (non-cumulative; repeatable)
social-gopher -profile developer alice
social-gopher -profile developer -profile community alice
social-gopher -profile full alice

# Verbose + CSV
social-gopher -v -csv alice

# Limit to specific sites (case-insensitive names)
social-gopher -site GitHub -site GitLab alice

# Route through local Tor
social-gopher -tor alice

# Custom SOCKS/HTTP proxy
social-gopher -proxy socks5h://127.0.0.1:9050 alice
```

## Usage

```bash
social-gopher [flags] <username>
social-gopher -validate-catalog [flags]
```

`-validate-catalog` takes no username. It probes each filtered site’s
`username_claimed` / `username_unclaimed` fixtures instead.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-csv` | off | Write found results to a sanitized `{username}.csv` in the current directory |
| `-tor` | off | Use Tor (reuse existing SOCKS or start system `tor`) |
| `-proxy` | | Proxy URL (`socks5h://`, `socks5://`, `http://`, `https://`); SOCKS schemes resolve DNS through the proxy |
| `-timeout` | `20s` | Per-site probe budget (shared by HEAD and any GET retry) |
| `-workers` | `20` | Concurrent workers (must be ≥ 1) |
| `-profile` | (default only) | Repeatable: `default`, `developer`, `creative`, `community`, or `full`. Omitted means default-profile sites only |
| `-site` | all | Repeatable site name filter (case-insensitive) |
| `-nsfw` | off | Include NSFW-tagged sites |
| `-v`, `-verbose` | off | Print non-hits, unknowns, invalid usernames, and errors |
| `-catalog` | bundled | Path to a custom `sites.json` (trust boundary: only load files you trust) |
| `-validate-catalog` | off | Run catalog self-tests (no scan username) |

`-profile` selects site groups as `default` plus each named profile
(non-cumulative; repeatable). `full` scans the entire catalog. Profiles are
applied before `-site` / `-nsfw`.

If both `-tor` and `-proxy` are set, `-proxy` wins and a notice is printed.

## Networking

Social Gopher does **not** honor `HTTP_PROXY`, `HTTPS_PROXY`, or `ALL_PROXY`.
Only `-proxy` or `-tor` changes egress.

### Direct

Default mode uses a plain HTTP client with no environment proxy.

### Proxy

```bash
social-gopher -proxy socks5h://127.0.0.1:9050 alice
```

Supported schemes: `socks5h`, `socks5`, `http`, `https`. Both SOCKS schemes
send DNS through the proxy (curl’s `socks5h` semantics). Prefer `socks5h` in
URLs.

### Tor

1. Install Tor so the `tor` binary is on your `PATH` (e.g. `apt install tor`,
   Homebrew).
2. Run with `-tor`. If a SOCKS5 proxy is already responding on
   `127.0.0.1:9050`, that listener is reused; otherwise Social Gopher starts a
   temporary Tor process and stops it when the scan finishes. A non-SOCKS
   process on that port is an error.
3. Or skip process management with
   `-proxy socks5h://127.0.0.1:9050`.

First bootstrap of a freshly started Tor can take ~30–60s. Tor exits are often
rate-limited or blocked by WAFs, so coverage may drop compared to a direct
scan.

## Understanding results

Each site probe is classified as one of:

| Result | Meaning |
|--------|---------|
| Found | Username appears present |
| Not found | Username appears absent |
| Invalid | Username fails the site’s `username_pattern` (no HTTP) |
| Unknown | Ambiguous HTTP outcome (for example first-response 3xx on status/body checks) |
| Error | Transport, body-read, or other probe failure |

Probes **never follow redirects**; classification uses the first response.

### Terminal output

- **Found** (always): green `[+] Site: url`
- **Summary** (always): `Found x/y profiles in <duration>`
- With **`-v`**: `[-]` not found / invalid username, `[?]` unknown, `[!]` errors
- **Spinner**: ephemeral checking status on a TTY only

Colors and the spinner are enabled only when stdout is a terminal.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success (scan finished, or validate with no failures) |
| `1` | Load, filter, transport, Tor, scanner, CSV, or validate failure |
| `2` | Usage / flag errors |
| `130` | Interrupted (`Ctrl+C` / SIGTERM) after a partial run |

## CSV export

`-csv` writes found rows only to a sanitized `{username}.csv` in the current
working directory (`/`, `\`, and NUL become `_`).

Columns:

`username,name,home_url,profile_url,exists,http_status,response_time_s`

Cells starting with `=`, `+`, `-`, `@`, tab, or CR are prefixed with `'` to
avoid spreadsheet formula injection.

## Site catalog

The default catalog is embedded from [`data/sites.json`](data/sites.json)
(445 sites across developer, creative, music, gaming, finance, regional, and
niche communities). Add sites by editing that file; no code changes are
required. Rebuild so the embed refreshes:

```bash
go build -o social-gopher ./cmd/social-gopher
```

Grow coverage in verified batches with `-validate-catalog`.

Custom `-catalog` files are a trust boundary. Load rejects unknown JSON
fields, non-`http`/`https` URLs, private/link-local/metadata hosts, `POST`,
dangerous hop-by-hop/`Host` headers, and invalid or over-long
`username_pattern` values. DNS rebinding is a residual risk if a hostname
later resolves to a private address.

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

`profile` is required and is the introducing wave for the site: `default`,
`developer`, `creative`, or `community`.

| `check.type` | Meaning |
|--------------|---------|
| `status` | Does not follow redirects; exists on first-response 2xx unless status is in `not_found_status` (default `[404]`) or optional `not_found_text` matches the body (soft-404; forces GET); other codes (including 3xx) are unknown |
| `body` | Does not follow redirects; missing if the first-response body contains any `not_found_text` substring; else exists on 2xx; other codes are unknown |
| `redirect` | Does not follow redirects; exists on 2xx, otherwise missing |

Use `{username}` in `profile_url`. When `probe_url` is set, it must also
contain `{username}` (use it when the check URL differs from the public
profile link).

`method`, when set, must be `GET` or `HEAD` (default depends on check type).

`not_found_text` is required for `body` checks and optional on `status` as a
soft-404 guard.

`username_claimed` / `username_unclaimed` are optional self-test fixtures
(ignored during normal scans).

### Verifying a new site

1. Add the site row with a working `check` type.
2. Set `username_claimed` (known existing account) and `username_unclaimed`
   (guaranteed missing).
3. Validate:

```bash
social-gopher -validate-catalog -site YourSite
```

4. Expect `PASS YourSite`. Sites missing self-test fields are reported as
   `SKIP`.
5. Rebuild so the embedded catalog picks up changes.

To validate every filtered site that has fixtures:

```bash
social-gopher -validate-catalog
```

## Project layout

```
cmd/social-gopher/     CLI composition root
internal/catalog/      Site catalog load + filter
internal/scan/         Concurrent probes + classifiers
internal/report/       Terminal + CSV writers
internal/transport/    HTTP client, Tor/proxy
internal/torrun/       Optional system Tor process lifecycle
data/                  Embedded catalog (embed.go + sites.json)
```

## Development

Local CI-parity checks:

```bash
# install: https://golangci-lint.run/docs/welcome/install/
golangci-lint fmt ./...
golangci-lint run ./...
go test ./... -race -count=1
```

Pull requests run build, `go test ./... -race`, govulncheck, and golangci-lint.
A scheduled workflow also runs `-validate-catalog -profile full`.

For agent and contributor conventions (rules, skills, architecture), see
[AGENTS.md](AGENTS.md).

## Ethics

Use only for lawful purposes and with permission where required. Respect site
terms of service and local law. This tool is for defensive OSINT and research,
not harassment or unauthorized access.

## License

MIT. See [LICENSE](LICENSE).
