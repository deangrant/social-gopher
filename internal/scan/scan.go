// Package scan probes catalog sites for a username concurrently.
package scan

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/deangrant/social-gopher/internal/catalog"
)

// Existence is the probe classification for a username on a site.
type Existence string

const (
	Found      Existence = "found"
	NotFound   Existence = "not_found"
	Invalid    Existence = "invalid"
	Unknown    Existence = "unknown"
	ErrorState Existence = "error"
)

// Result is the outcome of probing one site.
type Result struct {
	Site         catalog.Site
	Username     string
	ProfileURL   string
	Exists       Existence
	HTTPStatus   int
	ResponseTime time.Duration
	Err          error
}

// Options configures a Scanner.
type Options struct {
	// Client is used for probes. Redirects are disabled via CheckRedirect.
	Client  *http.Client
	Workers int
	// Timeout is the per-site probe budget shared by HEAD and any GET retry.
	// If unset or non-positive, defaults to 20s.
	Timeout time.Duration
	// OnStart is called when a site probe begins. Optional.
	OnStart func(site catalog.Site)
}

// Scanner runs concurrent username probes.
type Scanner struct {
	client  *http.Client
	workers int
	timeout time.Duration
	onStart func(site catalog.Site)
}

// New returns a Scanner. Client must be non-nil.
func New(opts Options) (*Scanner, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("client is required")
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = 20
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Scanner{
		client:  opts.Client,
		workers: workers,
		timeout: timeout,
		onStart: opts.OnStart,
	}, nil
}

// Run probes every site for username and sends results on the returned channel.
// The channel is closed when all work finishes or ctx is cancelled.
func (s *Scanner) Run(
	ctx context.Context,
	username string,
	sites []catalog.Site,
) <-chan Result {
	out := make(chan Result)
	jobs := make(chan catalog.Site)

	var wg sync.WaitGroup
	workers := s.workers
	if workers > len(sites) && len(sites) > 0 {
		workers = len(sites)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for site := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				res := s.Check(ctx, username, site)
				select {
				case out <- res:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, site := range sites {
			select {
			case jobs <- site:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Check probes a single site for username.
func (s *Scanner) Check(
	ctx context.Context,
	username string,
	site catalog.Site,
) Result {
	if s.onStart != nil {
		s.onStart(site)
	}
	profileURL := expandURL(site.ProfileURL, username)
	res := Result{
		Site:       site,
		Username:   username,
		ProfileURL: profileURL,
		Exists:     Unknown,
	}

	if site.UsernamePattern != "" {
		ok, err := site.UsernameMatches(username)
		if err != nil {
			res.Exists = ErrorState
			res.Err = err
			return res
		}
		if !ok {
			res.Exists = Invalid
			return res
		}
	}

	probeURL := profileURL
	if site.ProbeURL != "" {
		probeURL = expandURL(site.ProbeURL, username)
	}

	method := site.Method
	if site.Check.Type == catalog.CheckStatus &&
		len(site.Check.NotFoundText) > 0 {
		// Soft-404 text requires a response body.
		method = http.MethodGet
	} else if method == "" {
		if site.Check.Type == catalog.CheckStatus {
			method = http.MethodHead
		} else {
			method = http.MethodGet
		}
	}

	probeCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, method, probeURL, nil)
	if err != nil {
		res.Exists = ErrorState
		res.Err = err
		return res
	}
	for k, v := range site.Headers {
		req.Header.Set(k, v)
	}

	client := noRedirectClient(s.client)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// HEAD is often blocked; retry GET for status checks.
		if method == http.MethodHead &&
			site.Check.Type == catalog.CheckStatus &&
			probeCtx.Err() == nil {
			req, reqErr := http.NewRequestWithContext(
				probeCtx,
				http.MethodGet,
				probeURL,
				nil,
			)
			if reqErr == nil {
				for k, v := range site.Headers {
					req.Header.Set(k, v)
				}
				resp, err = client.Do(req)
			}
		}
	}
	res.ResponseTime = time.Since(start)
	if err != nil {
		res.Exists = ErrorState
		res.Err = err
		return res
	}
	defer resp.Body.Close()
	res.HTTPStatus = resp.StatusCode

	var body string
	if needsBody(site) {
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr != nil && site.Check.Type == catalog.CheckBody {
			res.Exists = ErrorState
			res.Err = readErr
			return res
		}
		body = string(raw)
	}

	res.Exists = classify(site, resp.StatusCode, body)
	return res
}

func needsBody(site catalog.Site) bool {
	return site.Check.Type == catalog.CheckBody ||
		len(site.Check.NotFoundText) > 0
}

func classify(site catalog.Site, status int, body string) Existence {
	switch site.Check.Type {
	case catalog.CheckStatus:
		for _, code := range site.Check.NotFoundStatus {
			if status == code {
				return NotFound
			}
		}
		if bodyHasNotFoundText(site.Check.NotFoundText, body) {
			return NotFound
		}
		if status >= 200 && status < 300 {
			return Found
		}
		return Unknown
	case catalog.CheckBody:
		if bodyHasNotFoundText(site.Check.NotFoundText, body) {
			return NotFound
		}
		if status >= 200 && status < 300 {
			return Found
		}
		return Unknown
	case catalog.CheckRedirect:
		if status >= 200 && status < 300 {
			return Found
		}
		return NotFound
	default:
		return Unknown
	}
}

func bodyHasNotFoundText(texts []string, body string) bool {
	if len(texts) == 0 {
		return false
	}
	lower := strings.ToLower(body)
	for _, text := range texts {
		if strings.Contains(lower, strings.ToLower(text)) {
			return true
		}
	}
	return false
}

func expandURL(tmpl, username string) string {
	return strings.ReplaceAll(tmpl, "{username}", urlPathEscape(username))
}

// urlPathEscape escapes username for path segments without
// turning "/" into %2F for the whole URL. Usernames should not
// contain slashes; we escape reserved chars.
func urlPathEscape(username string) string {
	var b strings.Builder
	for _, r := range username {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '~':
			b.WriteRune(r)
		default:
			for _, c := range []byte(string(r)) {
				fmt.Fprintf(&b, "%%%02X", c)
			}
		}
	}
	return b.String()
}

func noRedirectClient(c *http.Client) *http.Client {
	clone := *c
	clone.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &clone
}
