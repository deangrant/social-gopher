// Command social-gopher scans social networks and websites for a username.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/deangrant/social-gopher/data"
	"github.com/deangrant/social-gopher/internal/catalog"
	"github.com/deangrant/social-gopher/internal/report"
	"github.com/deangrant/social-gopher/internal/scan"
	"github.com/deangrant/social-gopher/internal/torrun"
	"github.com/deangrant/social-gopher/internal/transport"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("social-gopher", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		csvOut = fs.Bool(
			"csv",
			false,
			"write found results as CSV ({username}.csv)",
		)
		useTor = fs.Bool(
			"tor",
			false,
			"use Tor (start system tor if needed)",
		)
		proxyURL = fs.String(
			"proxy",
			"",
			"proxy URL (socks5h://, socks5://, http://, https://); "+
				"SOCKS schemes use remote DNS",
		)
		timeout = fs.Duration(
			"timeout",
			20*time.Second,
			"per-request timeout",
		)
		workers  = fs.Int("workers", 20, "number of concurrent workers")
		nsfw     = fs.Bool("nsfw", false, "include NSFW-tagged sites")
		verbose  = fs.Bool("v", false, "print non-hits and errors")
		catalogP = fs.String(
			"catalog",
			"",
			"path to sites catalog JSON (default: bundled data/sites.json)",
		)
		validateCatalog = fs.Bool(
			"validate-catalog",
			false,
			"probe username_claimed/unclaimed for catalog self-tests",
		)
	)
	var sites multiFlag
	var profiles multiFlag
	fs.Var(&sites, "site", "limit scan to site name (repeatable)")
	fs.Var(
		&profiles,
		"profile",
		"scan profile: default, developer, creative, community, full "+
			"(repeatable; default: default)",
	)
	fs.BoolVar(verbose, "verbose", false, "print non-hits and errors")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *validateCatalog {
		if fs.NArg() > 0 {
			fmt.Fprintln(
				os.Stderr,
				"usage: social-gopher -validate-catalog [flags]",
			)
			return 2
		}
	} else if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: social-gopher [flags] <username>")
		fs.PrintDefaults()
		return 2
	}

	allSites, err := loadCatalog(*catalogP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load catalog: %v\n", err)
		return 1
	}
	profiled, err := catalog.FilterByProfile(allSites, profiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}
	filtered, err := catalog.Filter(profiled, sites, *nsfw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "filter sites: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	trOpts := transport.Options{
		Timeout:  *timeout,
		ProxyURL: *proxyURL,
		UseTor:   *useTor,
	}
	if *useTor && *proxyURL == "" {
		ensureCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		inst, err := torrun.Ensure(ensureCtx, torrun.Options{})
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tor: %v\n", err)
			return 1
		}
		defer inst.Close()
		if inst.StartedByUs() {
			fmt.Fprintf(os.Stderr, "started Tor (pid %d)\n", inst.Pid())
		} else {
			fmt.Fprintln(os.Stderr, "using existing Tor")
		}
		trOpts.ProxyURL = inst.ProxyURL()
		trOpts.UseTor = false
	}

	tr, err := transport.New(trOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "transport: %v\n", err)
		return 1
	}
	if tr.Notice != "" {
		fmt.Fprintln(os.Stderr, tr.Notice)
	}

	if *validateCatalog {
		return runValidate(ctx, tr.Client, filtered, *timeout)
	}

	username := strings.TrimSpace(fs.Arg(0))
	if username == "" {
		fmt.Fprintln(os.Stderr, "username is required")
		return 2
	}

	printer := &report.Printer{
		Out:     os.Stdout,
		Verbose: *verbose,
		Color:   isTTY(os.Stdout),
	}

	scanner, err := scan.New(scan.Options{
		Client:  tr.Client,
		Workers: *workers,
		Timeout: *timeout,
		OnStart: func(site catalog.Site) {
			printer.Started(site.Name)
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "scanner: %v\n", err)
		return 1
	}

	start := time.Now()
	results := make([]scan.Result, 0, len(filtered))
	found := 0
	for r := range scanner.Run(ctx, username, filtered) {
		printer.Result(r)
		results = append(results, r)
		if r.Exists == scan.Found {
			found++
		}
	}
	checked := len(results)
	printer.Summary(found, checked, time.Since(start))

	if *csvOut {
		path, err := csvOutputPath(username)
		if err != nil {
			fmt.Fprintf(os.Stderr, "csv path: %v\n", err)
			return 1
		}
		if err := writeCSV(path, username, results); err != nil {
			fmt.Fprintf(os.Stderr, "write csv: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stdout, "Wrote %s\n", path)
	}

	if err := ctx.Err(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"scan interrupted after %d/%d sites: %v\n",
			checked,
			len(filtered),
			err,
		)
		return exitOnInterrupt(err)
	}
	return 0
}

func runValidate(
	ctx context.Context,
	client *http.Client,
	sites []catalog.Site,
	timeout time.Duration,
) int {
	scanner, err := scan.New(scan.Options{
		Client:  client,
		Workers: 1,
		Timeout: timeout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "scanner: %v\n", err)
		return 1
	}

	passed, failed, skipped := 0, 0, 0
	for _, site := range sites {
		if site.UsernameClaimed == "" || site.UsernameUnclaimed == "" {
			fmt.Fprintf(
				os.Stdout,
				"SKIP %s (missing username_claimed/unclaimed)\n",
				site.Name,
			)
			skipped++
			continue
		}

		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		claimed := scanner.Check(checkCtx, site.UsernameClaimed, site)
		cancel()

		checkCtx, cancel = context.WithTimeout(ctx, timeout)
		unclaimed := scanner.Check(checkCtx, site.UsernameUnclaimed, site)
		cancel()

		ok := true
		if claimed.Exists != scan.Found {
			ok = false
			fmt.Fprintf(
				os.Stdout,
				"FAIL %s claimed %q => %s",
				site.Name,
				site.UsernameClaimed,
				claimed.Exists,
			)
			if claimed.Err != nil {
				fmt.Fprintf(os.Stdout, " (%v)", claimed.Err)
			}
			fmt.Fprintf(os.Stdout, " HTTP %d\n", claimed.HTTPStatus)
		}
		if unclaimed.Exists != scan.NotFound {
			ok = false
			fmt.Fprintf(
				os.Stdout,
				"FAIL %s unclaimed %q => %s",
				site.Name,
				site.UsernameUnclaimed,
				unclaimed.Exists,
			)
			if unclaimed.Err != nil {
				fmt.Fprintf(os.Stdout, " (%v)", unclaimed.Err)
			}
			fmt.Fprintf(os.Stdout, " HTTP %d\n", unclaimed.HTTPStatus)
		}
		if ok {
			fmt.Fprintf(os.Stdout, "PASS %s\n", site.Name)
			passed++
		} else {
			failed++
		}

		if ctx.Err() != nil {
			break
		}
	}

	fmt.Fprintf(
		os.Stdout,
		"\nvalidate: %d passed, %d failed, %d skipped\n",
		passed,
		failed,
		skipped,
	)
	if err := ctx.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "validate interrupted: %v\n", err)
		return exitOnInterrupt(err)
	}
	if failed > 0 {
		return 1
	}
	return 0
}

// exitOnInterrupt maps a context error after an incomplete run to a process
// exit code. Canceled (Ctrl+C / SIGTERM via NotifyContext) uses 130.
func exitOnInterrupt(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	return 1
}

// csvOutputPath returns a cwd-relative CSV filename derived from username.
// Path separators and NUL are replaced so the file cannot escape the
// working directory.
func csvOutputPath(username string) (string, error) {
	mapped := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', 0:
			return '_'
		default:
			return r
		}
	}, username)
	name := filepath.Base(mapped)
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("username is unsafe for csv filename")
	}
	return name + ".csv", nil
}

func writeCSV(path, username string, results []scan.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return report.WriteCSV(f, username, results, true)
}

func loadCatalog(explicit string) ([]catalog.Site, error) {
	if explicit != "" {
		return catalog.LoadFile(explicit)
	}
	return catalog.Load(bytes.NewReader(data.SitesJSON))
}

func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// multiFlag collects repeated -site values.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}
