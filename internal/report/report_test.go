package report_test

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deangrant/social-gopher/internal/catalog"
	"github.com/deangrant/social-gopher/internal/report"
	"github.com/deangrant/social-gopher/internal/scan"
)

func TestWriteCSV(t *testing.T) {
	results := []scan.Result{
		{
			Site: catalog.Site{
				Name:    "GitHub",
				HomeURL: "https://github.com",
			},
			ProfileURL:   "https://github.com/octocat",
			Exists:       scan.Found,
			HTTPStatus:   200,
			ResponseTime: 1500 * time.Millisecond,
		},
		{
			Site: catalog.Site{
				Name:    "Missing",
				HomeURL: "https://m.test",
			},
			ProfileURL: "https://m.test/x",
			Exists:     scan.NotFound,
			HTTPStatus: 404,
		},
	}
	var buf bytes.Buffer
	if err := report.WriteCSV(&buf, "octocat", results, true); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf(
			"lines = %d, want 2 (header + found only)\n%s",
			len(lines),
			buf.String(),
		)
	}
	if !strings.Contains(
		lines[0],
		"username,name,home_url,profile_url,exists,http_status,response_time_s",
	) {
		t.Fatalf("header = %q", lines[0])
	}
	if !strings.Contains(lines[1], "octocat,GitHub,") {
		t.Fatalf("row = %q", lines[1])
	}

	buf.Reset()
	if err := report.WriteCSV(&buf, "octocat", results, false); err != nil {
		t.Fatal(err)
	}
	all := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(all) != 3 {
		t.Fatalf(
			"foundOnly=false lines = %d, want 3\n%s",
			len(all),
			buf.String(),
		)
	}
	if !strings.Contains(all[2], "Missing") {
		t.Fatalf("missing not-found row: %q", buf.String())
	}
}

func TestWriteCSVFormulaInjection(t *testing.T) {
	results := []scan.Result{
		{
			Site: catalog.Site{
				Name:    "+cmd",
				HomeURL: "=http://evil.example",
			},
			ProfileURL:   "@https://evil.example/x",
			Exists:       scan.Found,
			HTTPStatus:   200,
			ResponseTime: time.Second,
		},
	}
	var buf bytes.Buffer
	if err := report.WriteCSV(&buf, "=1+1", results, true); err != nil {
		t.Fatal(err)
	}
	row := strings.Split(strings.TrimSpace(buf.String()), "\n")[1]
	wants := []string{
		"'=1+1",
		"'+cmd",
		"'=http://evil.example",
		"'@https://evil.example/x",
	}
	for _, want := range wants {
		if !strings.Contains(row, want) {
			t.Fatalf("row %q missing sanitized %q", row, want)
		}
	}
	if strings.Contains(row, ",=1+1,") || strings.HasPrefix(row, "=1+1,") {
		t.Fatalf("unsanitized username in row %q", row)
	}
}

func TestPrinterColorOffFoundOnly(t *testing.T) {
	var buf bytes.Buffer
	p := &report.Printer{Out: &buf, Color: false}
	p.Started("GitHub")
	p.Result(scan.Result{
		Site:       catalog.Site{Name: "GitHub"},
		ProfileURL: "https://github.com/x",
		Exists:     scan.Found,
	})
	p.Result(scan.Result{
		Site:   catalog.Site{Name: "Gone"},
		Exists: scan.NotFound,
	})
	p.Summary(1, 2, time.Second)

	out := buf.String()
	if strings.Contains(out, "Checking") {
		t.Fatalf("color-off Started should not print status, got %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Fatalf("color-off output should have no ANSI, got %q", out)
	}
	if !strings.Contains(out, "[+] GitHub: https://github.com/x") {
		t.Fatalf("missing found line: %q", out)
	}
	if strings.Contains(out, "Gone") {
		t.Fatalf("not-found should be silent without verbose: %q", out)
	}
	if !strings.Contains(out, "Found 1/2") {
		t.Fatalf("missing summary: %q", out)
	}
}

func TestPrinterColorOnStatusAndFound(t *testing.T) {
	var buf bytes.Buffer
	p := &report.Printer{Out: &buf, Color: true}
	p.Started("Instagram")
	// Stop spinner before asserting so ticks do not race the buffer.
	p.Summary(0, 1, 0)
	out := buf.String()
	statusRe := regexp.MustCompile(`\[[|/\-]\] Checking Instagram`)
	if !strings.Contains(out, "\r") || !statusRe.MatchString(out) {
		t.Fatalf("Started status = %q, want spinner Checking Instagram", out)
	}
	if !strings.Contains(out, "\033[37m") {
		t.Fatalf("status should be white: %q", out)
	}
}

func TestPrinterColorOnFoundLine(t *testing.T) {
	var buf bytes.Buffer
	p := &report.Printer{Out: &buf, Color: true}
	p.Started("Instagram")
	p.Result(scan.Result{
		Site:       catalog.Site{Name: "Instagram"},
		ProfileURL: "https://instagram.com/x",
		Exists:     scan.Found,
	})
	p.Summary(1, 1, time.Second)
	out := buf.String()
	if !strings.Contains(
		out,
		"\033[32m[+] Instagram:\033[0m https://instagram.com/x",
	) {
		t.Fatalf("found line = %q, want green label", out)
	}
}

func TestSpinnerCharCycles(t *testing.T) {
	frames := []byte{'|', '/', '-', '\\'}
	for i, want := range frames {
		if got := report.SpinnerCharForTest(i); got != want {
			t.Fatalf("SpinnerCharForTest(%d) = %q, want %q", i, got, want)
		}
	}
	if got := report.SpinnerCharForTest(4); got != '|' {
		t.Fatalf("wrap = %q, want |", got)
	}
}

func TestPrinterVerboseNotFound(t *testing.T) {
	var buf bytes.Buffer
	p := &report.Printer{Out: &buf, Verbose: true, Color: false}
	p.Started("Gone")
	p.Result(scan.Result{
		Site:   catalog.Site{Name: "Gone"},
		Exists: scan.NotFound,
	})
	out := buf.String()
	if !strings.Contains(out, "[-] Gone: not found") {
		t.Fatalf("verbose not-found missing: %q", out)
	}
}

func TestPrinterConcurrentStartedResult(_ *testing.T) {
	p := &report.Printer{Out: io.Discard, Color: true}
	var wg sync.WaitGroup
	const n = 50
	for i := 0; i < n; i++ {
		wg.Add(2)
		name := fmt.Sprintf("Site%d", i)
		go func(site string) {
			defer wg.Done()
			p.Started(site)
		}(name)
		go func(site string) {
			defer wg.Done()
			p.Result(scan.Result{
				Site:       catalog.Site{Name: site},
				ProfileURL: "https://example.com/" + site,
				Exists:     scan.Found,
			})
		}(name)
	}
	wg.Wait()
	p.Summary(n, n, time.Millisecond)
}
