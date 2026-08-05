// Package report formats probe results for the terminal and
// optional CSV export.
package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deangrant/social-gopher/internal/scan"
)

const (
	ansiReset = "\033[0m"
	ansiGreen = "\033[32m"
	ansiWhite = "\033[37m"
	ansiClear = "\r\033[2K"

	spinnerInterval = 100 * time.Millisecond
)

var spinnerFrames = []byte{'|', '/', '-', '\\'}

// Printer writes live progress to an output stream.
// When Color is true (TTY), a single ephemeral white status line shows the
// current check with a rotating spinner; lasting output is green found lines
// (and verbose non-hits).
type Printer struct {
	Out     io.Writer
	Verbose bool
	Color   bool

	mu          sync.Mutex
	inflight    map[string]struct{}
	statusShown bool
	lastStatus  string
	spinIdx     int
	spinning    bool
	stopSpin    chan struct{}
}

// Started records that a site probe has begun and updates the status line.
func (p *Printer) Started(siteName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.inflight == nil {
		p.inflight = make(map[string]struct{})
	}
	p.inflight[siteName] = struct{}{}
	if !p.Color {
		return
	}
	p.renderStatusLocked(siteName)
	p.ensureSpinnerLocked()
}

// Result prints a completed probe according to verbosity and color mode.
func (p *Printer) Result(r scan.Result) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.inflight != nil {
		delete(p.inflight, r.Site.Name)
	}

	switch r.Exists {
	case scan.Found:
		p.clearStatusLocked()
		p.printFoundLocked(r.Site.Name, r.ProfileURL)
	case scan.NotFound:
		if p.Verbose {
			p.clearStatusLocked()
			fmt.Fprintf(p.Out, "[-] %s: not found\n", r.Site.Name)
		}
	case scan.Invalid:
		if p.Verbose {
			p.clearStatusLocked()
			fmt.Fprintf(
				p.Out,
				"[-] %s: username invalid for site\n",
				r.Site.Name,
			)
		}
	case scan.Unknown:
		if p.Verbose {
			p.clearStatusLocked()
			fmt.Fprintf(
				p.Out,
				"[?] %s: unknown (HTTP %d)\n",
				r.Site.Name,
				r.HTTPStatus,
			)
		}
	case scan.ErrorState:
		if p.Verbose {
			p.clearStatusLocked()
			errMsg := "error"
			if r.Err != nil {
				errMsg = r.Err.Error()
			}
			fmt.Fprintf(p.Out, "[!] %s: %s\n", r.Site.Name, errMsg)
		}
	}

	if p.Color && len(p.inflight) > 0 {
		p.renderStatusLocked(anyInflight(p.inflight))
		p.ensureSpinnerLocked()
	} else if p.Color && len(p.inflight) == 0 {
		p.clearStatusLocked()
		p.stopSpinnerLocked()
	}
}

// Summary writes a short end-of-run line.
func (p *Printer) Summary(found, total int, elapsed time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.clearStatusLocked()
	p.stopSpinnerLocked()
	fmt.Fprintf(
		p.Out,
		"\nFound %d/%d profiles in %s\n",
		found,
		total,
		elapsed.Round(time.Millisecond),
	)
}

func (p *Printer) printFoundLocked(name, profileURL string) {
	label := fmt.Sprintf("[+] %s:", name)
	if p.Color {
		fmt.Fprintf(
			p.Out,
			"%s%s%s %s\n",
			ansiGreen,
			label,
			ansiReset,
			profileURL,
		)
		return
	}
	fmt.Fprintf(p.Out, "%s %s\n", label, profileURL)
}

func (p *Printer) renderStatusLocked(siteName string) {
	frame := spinnerChar(p.spinIdx)
	line := fmt.Sprintf("[%c] Checking %s", frame, siteName)
	if p.Color {
		fmt.Fprintf(p.Out, "%s%s%s%s", ansiClear, ansiWhite, line, ansiReset)
	} else {
		fmt.Fprintf(p.Out, "%s%s", ansiClear, line)
	}
	p.flushOut()
	p.statusShown = true
	p.lastStatus = siteName
}

func (p *Printer) clearStatusLocked() {
	if !p.statusShown {
		return
	}
	fmt.Fprint(p.Out, ansiClear)
	p.flushOut()
	p.statusShown = false
	p.lastStatus = ""
}

func (p *Printer) ensureSpinnerLocked() {
	if p.spinning || !p.Color {
		return
	}
	p.spinning = true
	p.stopSpin = make(chan struct{})
	stop := p.stopSpin
	go p.runSpinner(stop)
}

func (p *Printer) stopSpinnerLocked() {
	if !p.spinning {
		return
	}
	close(p.stopSpin)
	p.spinning = false
	p.stopSpin = nil
}

func (p *Printer) runSpinner(stop <-chan struct{}) {
	t := time.NewTicker(spinnerInterval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			p.mu.Lock()
			if !p.spinning || len(p.inflight) == 0 {
				p.mu.Unlock()
				return
			}
			p.spinIdx++
			p.renderStatusLocked(anyInflight(p.inflight))
			p.mu.Unlock()
		}
	}
}

func (p *Printer) flushOut() {
	type flusher interface {
		Flush() error
	}
	if f, ok := p.Out.(flusher); ok {
		_ = f.Flush()
	}
}

func spinnerChar(idx int) byte {
	return spinnerFrames[idx%len(spinnerFrames)]
}

func anyInflight(m map[string]struct{}) string {
	for name := range m {
		return name
	}
	return ""
}

// WriteCSV writes all results (or only found, if foundOnly) as CSV.
func WriteCSV(
	w io.Writer,
	username string,
	results []scan.Result,
	foundOnly bool,
) error {
	cw := csv.NewWriter(w)
	header := []string{
		"username",
		"name",
		"home_url",
		"profile_url",
		"exists",
		"http_status",
		"response_time_s",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range results {
		if foundOnly && r.Exists != scan.Found {
			continue
		}
		row := []string{
			csvCell(username),
			csvCell(r.Site.Name),
			csvCell(r.Site.HomeURL),
			csvCell(r.ProfileURL),
			csvCell(string(r.Exists)),
			csvCell(strconv.Itoa(r.HTTPStatus)),
			csvCell(formatSeconds(r.ResponseTime)),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvCell prefixes formula-leading values so spreadsheet apps
// treat them as text.
func csvCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	default:
		return v
	}
}

func formatSeconds(d time.Duration) string {
	return strings.TrimRight(
		strings.TrimRight(fmt.Sprintf("%.3f", d.Seconds()), "0"),
		".",
	)
}
