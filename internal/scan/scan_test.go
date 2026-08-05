package scan_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deangrant/social-gopher/internal/catalog"
	"github.com/deangrant/social-gopher/internal/scan"
)

func TestClassifyStatus(t *testing.T) {
	var hit string
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit = r.URL.Path
			if strings.HasSuffix(r.URL.Path, "/missing") {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
	)
	t.Cleanup(srv.Close)

	sites := []catalog.Site{
		{
			Name:       "Ok",
			HomeURL:    srv.URL,
			ProfileURL: srv.URL + "/{username}",
			Method:     http.MethodGet,
			Check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
		},
		{
			Name:       "Missing",
			HomeURL:    srv.URL,
			ProfileURL: srv.URL + "/{username}",
			Method:     http.MethodGet,
			Check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
		},
	}

	sc, err := scan.New(scan.Options{Client: srv.Client(), Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got := map[string]scan.Existence{}
	for r := range sc.Run(ctx, "alice", []catalog.Site{sites[0]}) {
		got[r.Site.Name] = r.Exists
	}
	if got["Ok"] != scan.Found {
		t.Fatalf("alice exists = %v, want found (last path %s)", got["Ok"], hit)
	}

	for r := range sc.Run(ctx, "missing", []catalog.Site{sites[1]}) {
		got[r.Site.Name] = r.Exists
	}
	if got["Missing"] != scan.NotFound {
		t.Fatalf("missing exists = %v, want not_found", got["Missing"])
	}
}

func TestStatusSoft404NotFoundText(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if strings.HasSuffix(r.URL.Path, "/missing") {
				_, _ = w.Write([]byte("Sorry, user not found"))
				return
			}
			_, _ = w.Write([]byte("welcome to the profile"))
		}),
	)
	t.Cleanup(srv.Close)

	site := catalog.Site{
		Name:       "Soft404",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
			NotFoundText:   []string{"user not found"},
		},
	}
	sc, err := scan.New(scan.Options{Client: srv.Client(), Workers: 1})
	if err != nil {
		t.Fatal(err)
	}

	var missing, present scan.Existence
	for r := range sc.Run(
		context.Background(),
		"missing",
		[]catalog.Site{site},
	) {
		missing = r.Exists
	}
	if missing != scan.NotFound {
		t.Fatalf("missing exists = %v, want not_found", missing)
	}
	for r := range sc.Run(
		context.Background(),
		"alice",
		[]catalog.Site{site},
	) {
		present = r.Exists
	}
	if present != scan.Found {
		t.Fatalf("alice exists = %v, want found", present)
	}
}

func TestStatusSoft404ForcesGET(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
	)
	t.Cleanup(srv.Close)

	withText := catalog.Site{
		Name:       "WithText",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Method:     http.MethodHead,
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
			NotFoundText:   []string{"user not found"},
		},
	}
	sc, err := scan.New(scan.Options{Client: srv.Client(), Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	for range sc.Run(
		context.Background(),
		"alice",
		[]catalog.Site{withText},
	) {
		continue
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET when not_found_text set", gotMethod)
	}

	gotMethod = ""
	pure := catalog.Site{
		Name:       "PureStatus",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	for range sc.Run(
		context.Background(),
		"alice",
		[]catalog.Site{pure},
	) {
		continue
	}
	if gotMethod != http.MethodHead {
		t.Fatalf(
			"method = %q, want HEAD for status without not_found_text",
			gotMethod,
		)
	}
}

func TestClassifyBody(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if strings.Contains(r.URL.Path, "gone") {
				_, _ = w.Write([]byte("user does not exist here"))
				return
			}
			_, _ = w.Write([]byte("welcome back"))
		}),
	)
	t.Cleanup(srv.Close)

	site := catalog.Site{
		Name:       "Body",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Check: catalog.Check{
			Type:         catalog.CheckBody,
			NotFoundText: []string{"does not exist"},
		},
	}
	sc, err := scan.New(scan.Options{Client: srv.Client(), Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	var exists scan.Existence
	for r := range sc.Run(ctx, "gone", []catalog.Site{site}) {
		exists = r.Exists
	}
	if exists != scan.NotFound {
		t.Fatalf("exists = %v, want not_found", exists)
	}
	for r := range sc.Run(ctx, "alice", []catalog.Site{site}) {
		exists = r.Exists
	}
	if exists != scan.Found {
		t.Fatalf("exists = %v, want found", exists)
	}
}

func TestClassifyRedirect(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/gone") {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	t.Cleanup(srv.Close)

	site := catalog.Site{
		Name:       "Redir",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Check:      catalog.Check{Type: catalog.CheckRedirect},
	}
	sc, err := scan.New(scan.Options{Client: srv.Client(), Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	var exists scan.Existence
	for r := range sc.Run(ctx, "alice", []catalog.Site{site}) {
		exists = r.Exists
		if r.HTTPStatus != http.StatusOK {
			t.Fatalf("status = %d, want 200", r.HTTPStatus)
		}
	}
	if exists != scan.Found {
		t.Fatalf("exists = %v, want found", exists)
	}
	for r := range sc.Run(ctx, "gone", []catalog.Site{site}) {
		exists = r.Exists
	}
	if exists != scan.NotFound {
		t.Fatalf("exists = %v, want not_found", exists)
	}
}

func TestStatusDoesNotFollowRedirect(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/missing") {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			if r.URL.Path == "/login" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("please sign in"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("profile"))
		}),
	)
	t.Cleanup(srv.Close)

	site := catalog.Site{
		Name:       "StatusRedir",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Method:     http.MethodGet,
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	sc, err := scan.New(scan.Options{Client: srv.Client(), Workers: 1})
	if err != nil {
		t.Fatal(err)
	}

	var got scan.Result
	for r := range sc.Run(
		context.Background(),
		"missing",
		[]catalog.Site{site},
	) {
		got = r
	}
	if got.Exists == scan.Found {
		t.Fatal("exists = found; must not follow redirect to final 200")
	}
	if got.Exists != scan.Unknown {
		t.Fatalf("exists = %v, want unknown", got.Exists)
	}
	if got.HTTPStatus != http.StatusFound {
		t.Fatalf("status = %d, want 302", got.HTTPStatus)
	}
}

func TestBodyDoesNotFollowRedirect(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/missing") {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			if r.URL.Path == "/login" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("please sign in"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("welcome back"))
		}),
	)
	t.Cleanup(srv.Close)

	site := catalog.Site{
		Name:       "BodyRedir",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Check: catalog.Check{
			Type:         catalog.CheckBody,
			NotFoundText: []string{"does not exist"},
		},
	}
	sc, err := scan.New(scan.Options{Client: srv.Client(), Workers: 1})
	if err != nil {
		t.Fatal(err)
	}

	var got scan.Result
	for r := range sc.Run(
		context.Background(),
		"missing",
		[]catalog.Site{site},
	) {
		got = r
	}
	if got.Exists == scan.Found {
		t.Fatal("exists = found; must not follow redirect to final 200")
	}
	if got.Exists != scan.Unknown {
		t.Fatalf("exists = %v, want unknown", got.Exists)
	}
	if got.HTTPStatus != http.StatusFound {
		t.Fatalf("status = %d, want 302", got.HTTPStatus)
	}
}

func TestUsernamePatternInvalid(t *testing.T) {
	site := catalog.Site{
		Name:            "Pat",
		HomeURL:         "https://example.com",
		ProfileURL:      "https://example.com/{username}",
		UsernamePattern: `^[a-z]+$`,
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	sc, err := scan.New(scan.Options{Client: http.DefaultClient, Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	for r := range sc.Run(
		context.Background(),
		"Bad_User",
		[]catalog.Site{site},
	) {
		if r.Exists != scan.Invalid {
			t.Fatalf("exists = %v, want invalid", r.Exists)
		}
	}
}

func TestUsernamePatternLoaded(t *testing.T) {
	const src = `{
		"sites": [{
			"name": "Pat",
			"home_url": "https://example.com",
			"profile_url": "https://example.com/{username}",
			"username_pattern": "^[a-z]+$",
			"check": { "type": "status", "not_found_status": [404] },
			"profile": "default"
		}]
	}`
	sites, err := catalog.Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	sc, err := scan.New(scan.Options{Client: http.DefaultClient, Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	for r := range sc.Run(context.Background(), "Bad_User", sites) {
		if r.Exists != scan.Invalid {
			t.Fatalf("exists = %v, want invalid", r.Exists)
		}
	}
}

func TestOnStart(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	t.Cleanup(srv.Close)

	started := make(chan string, 1)
	site := catalog.Site{
		Name:       "S",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Method:     http.MethodGet,
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	sc, err := scan.New(scan.Options{
		Client:  srv.Client(),
		Workers: 1,
		OnStart: func(s catalog.Site) { started <- s.Name },
	})
	if err != nil {
		t.Fatal(err)
	}
	for range sc.Run(context.Background(), "u", []catalog.Site{site}) {
		continue
	}
	select {
	case name := <-started:
		if name != "S" {
			t.Fatalf("started = %q", name)
		}
	default:
		t.Fatal("OnStart was not called")
	}
}

type seqTransport struct {
	mu    sync.Mutex
	calls []string
	fn    func(req *http.Request, n int) (*http.Response, error)
}

func (t *seqTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	n := len(t.calls)
	t.calls = append(t.calls, req.Method)
	fn := t.fn
	t.mu.Unlock()
	return fn(req, n)
}

func (t *seqTransport) methods() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.calls))
	copy(out, t.calls)
	return out
}

func TestSharedProbeTimeoutSkipsGETRetry(t *testing.T) {
	tr := &seqTransport{
		fn: func(req *http.Request, _ int) (*http.Response, error) {
			if req.Method != http.MethodHead {
				t.Errorf("unexpected method %s", req.Method)
				return nil, errors.New("unexpected GET")
			}
			<-req.Context().Done()
			return nil, req.Context().Err()
		},
	}
	site := catalog.Site{
		Name:       "SlowHEAD",
		HomeURL:    "https://example.com",
		ProfileURL: "https://example.com/{username}",
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	timeout := 50 * time.Millisecond
	sc, err := scan.New(scan.Options{
		Client:  &http.Client{Transport: tr},
		Workers: 1,
		Timeout: timeout,
	})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	res := sc.Check(context.Background(), "alice", site)
	elapsed := time.Since(start)

	if res.Exists != scan.ErrorState {
		t.Fatalf("exists = %v, want error", res.Exists)
	}
	methods := tr.methods()
	if len(methods) != 1 || methods[0] != http.MethodHead {
		t.Fatalf("methods = %v, want [HEAD] only", methods)
	}
	if elapsed > 3*timeout {
		t.Fatalf(
			"elapsed %v exceeds ~1x timeout %v (got ~2x?)",
			elapsed,
			timeout,
		)
	}
	if res.ResponseTime > 3*timeout {
		t.Fatalf(
			"ResponseTime %v exceeds ~1x timeout %v",
			res.ResponseTime,
			timeout,
		)
	}
}

func TestFastHEADFailureStillRetriesGET(t *testing.T) {
	tr := &seqTransport{
		fn: func(req *http.Request, _ int) (*http.Response, error) {
			if req.Method == http.MethodHead {
				return nil, errors.New("head blocked")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		},
	}
	site := catalog.Site{
		Name:       "RetryGET",
		HomeURL:    "https://example.com",
		ProfileURL: "https://example.com/{username}",
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	sc, err := scan.New(scan.Options{
		Client:  &http.Client{Transport: tr},
		Workers: 1,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	res := sc.Check(context.Background(), "alice", site)
	if res.Exists != scan.Found {
		t.Fatalf("exists = %v, want found (err=%v)", res.Exists, res.Err)
	}
	methods := tr.methods()
	if len(methods) != 2 ||
		methods[0] != http.MethodHead ||
		methods[1] != http.MethodGet {
		t.Fatalf("methods = %v, want [HEAD GET]", methods)
	}
}

func TestNoRedirectOverridesFollowingClient(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/missing") {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			if r.URL.Path == "/login" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("please sign in"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("profile"))
		}),
	)
	t.Cleanup(srv.Close)

	following := &http.Client{
		Transport: srv.Client().Transport,
		// Default CheckRedirect follows redirects.
	}
	site := catalog.Site{
		Name:       "FollowOverride",
		HomeURL:    srv.URL,
		ProfileURL: srv.URL + "/{username}",
		Method:     http.MethodGet,
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	sc, err := scan.New(scan.Options{Client: following, Workers: 1})
	if err != nil {
		t.Fatal(err)
	}

	got := sc.Check(context.Background(), "missing", site)
	if got.Exists == scan.Found {
		t.Fatal("exists = found; must not follow redirect to final 200")
	}
	if got.Exists != scan.Unknown {
		t.Fatalf("exists = %v, want unknown", got.Exists)
	}
	if got.HTTPStatus != http.StatusFound {
		t.Fatalf("status = %d, want 302", got.HTTPStatus)
	}
}

type countingReadCloser struct {
	r io.ReadCloser
	n *int
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	nr, err := c.r.Read(p)
	*c.n += nr
	return nr, err
}

func (c *countingReadCloser) Close() error {
	return c.r.Close()
}

func TestStatusCheckSkipsBodyRead(t *testing.T) {
	var readBytes int
	payload := strings.Repeat("x", 1<<20)
	tr := &seqTransport{
		fn: func(req *http.Request, _ int) (*http.Response, error) {
			body := &countingReadCloser{
				r: io.NopCloser(strings.NewReader(payload)),
				n: &readBytes,
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	}
	site := catalog.Site{
		Name:       "StatusOnly",
		HomeURL:    "https://example.com",
		ProfileURL: "https://example.com/{username}",
		Method:     http.MethodGet,
		Check: catalog.Check{
			Type:           catalog.CheckStatus,
			NotFoundStatus: []int{404},
		},
	}
	sc, err := scan.New(scan.Options{
		Client:  &http.Client{Transport: tr},
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := sc.Check(context.Background(), "alice", site)
	if got.Exists != scan.Found {
		t.Fatalf("exists = %v, want found", got.Exists)
	}
	if readBytes != 0 {
		t.Fatalf("body bytes read = %d, want 0", readBytes)
	}
}

func TestBodyCheckStillReadsBody(t *testing.T) {
	var readBytes int
	tr := &seqTransport{
		fn: func(req *http.Request, _ int) (*http.Response, error) {
			body := &countingReadCloser{
				r: io.NopCloser(strings.NewReader("user does not exist")),
				n: &readBytes,
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	}
	site := catalog.Site{
		Name:       "BodyNeeded",
		HomeURL:    "https://example.com",
		ProfileURL: "https://example.com/{username}",
		Check: catalog.Check{
			Type:         catalog.CheckBody,
			NotFoundText: []string{"does not exist"},
		},
	}
	sc, err := scan.New(scan.Options{
		Client:  &http.Client{Transport: tr},
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := sc.Check(context.Background(), "alice", site)
	if got.Exists != scan.NotFound {
		t.Fatalf("exists = %v, want not_found", got.Exists)
	}
	if readBytes == 0 {
		t.Fatal("body bytes read = 0, want > 0 for body check")
	}
}

func TestClassifyMatrix(t *testing.T) {
	// Scanner only accepts *http.Client (no separate Doer); redirect
	// first-response is covered via httptest + noRedirectClient clone.
	tests := []struct {
		name       string
		check      catalog.Check
		method     string
		status     int
		body       string
		bodyErr    error
		redirectTo string
		want       scan.Existence
		wantStatus int
	}{
		{
			name: "status_200",
			check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
			method:     http.MethodGet,
			status:     http.StatusOK,
			body:       "ok",
			want:       scan.Found,
			wantStatus: 200,
		},
		{
			name: "status_404",
			check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
			method:     http.MethodGet,
			status:     http.StatusNotFound,
			want:       scan.NotFound,
			wantStatus: 404,
		},
		{
			name: "status_403",
			check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
			method:     http.MethodGet,
			status:     http.StatusForbidden,
			want:       scan.Unknown,
			wantStatus: 403,
		},
		{
			name: "status_429",
			check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
			method:     http.MethodGet,
			status:     http.StatusTooManyRequests,
			want:       scan.Unknown,
			wantStatus: 429,
		},
		{
			name: "status_302_first_response",
			check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
			method:     http.MethodGet,
			redirectTo: "/login-ok",
			want:       scan.Unknown,
			wantStatus: http.StatusFound,
		},
		{
			name: "status_soft404_match",
			check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
				NotFoundText:   []string{"not found"},
			},
			method:     http.MethodGet,
			status:     http.StatusOK,
			body:       "user not found here",
			want:       scan.NotFound,
			wantStatus: 200,
		},
		{
			name: "status_soft404_no_match",
			check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
				NotFoundText:   []string{"not found"},
			},
			method:     http.MethodGet,
			status:     http.StatusOK,
			body:       "welcome profile",
			want:       scan.Found,
			wantStatus: 200,
		},
		{
			name: "body_200_found",
			check: catalog.Check{
				Type:         catalog.CheckBody,
				NotFoundText: []string{"missing"},
			},
			method:     http.MethodGet,
			status:     http.StatusOK,
			body:       "hello",
			want:       scan.Found,
			wantStatus: 200,
		},
		{
			name: "body_404_unknown",
			check: catalog.Check{
				Type:         catalog.CheckBody,
				NotFoundText: []string{"missing"},
			},
			method:     http.MethodGet,
			status:     http.StatusNotFound,
			body:       "nope",
			want:       scan.Unknown,
			wantStatus: 404,
		},
		{
			name: "body_302_first_response",
			check: catalog.Check{
				Type:         catalog.CheckBody,
				NotFoundText: []string{"missing"},
			},
			method:     http.MethodGet,
			redirectTo: "/login-ok",
			want:       scan.Unknown,
			wantStatus: http.StatusFound,
		},
		{
			name: "body_read_error",
			check: catalog.Check{
				Type:         catalog.CheckBody,
				NotFoundText: []string{"missing"},
			},
			method:  http.MethodGet,
			status:  http.StatusOK,
			bodyErr: errors.New("truncated"),
			want:    scan.ErrorState,
		},
		{
			name:       "redirect_200_found",
			check:      catalog.Check{Type: catalog.CheckRedirect},
			method:     http.MethodGet,
			status:     http.StatusOK,
			want:       scan.Found,
			wantStatus: 200,
		},
		{
			name:       "redirect_302_not_found",
			check:      catalog.Check{Type: catalog.CheckRedirect},
			method:     http.MethodGet,
			redirectTo: "/elsewhere",
			want:       scan.NotFound,
			wantStatus: http.StatusFound,
		},
		{
			name:       "redirect_404_not_found",
			check:      catalog.Check{Type: catalog.CheckRedirect},
			method:     http.MethodGet,
			status:     http.StatusNotFound,
			want:       scan.NotFound,
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followed := false
			var tr http.RoundTripper
			if tt.redirectTo != "" {
				srv := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path == tt.redirectTo {
							followed = true
							w.WriteHeader(http.StatusOK)
							return
						}
						http.Redirect(w, r, tt.redirectTo, http.StatusFound)
					}),
				)
				t.Cleanup(srv.Close)
				tr = srv.Client().Transport
				site := catalog.Site{
					Name:       tt.name,
					HomeURL:    srv.URL,
					ProfileURL: srv.URL + "/{username}",
					Method:     tt.method,
					Check:      tt.check,
				}
				sc, err := scan.New(scan.Options{
					Client:  &http.Client{Transport: tr},
					Workers: 1,
				})
				if err != nil {
					t.Fatal(err)
				}
				got := sc.Check(context.Background(), "alice", site)
				if got.Exists != tt.want {
					t.Fatalf("exists = %v, want %v", got.Exists, tt.want)
				}
				if got.HTTPStatus != tt.wantStatus {
					t.Fatalf(
						"status = %d, want %d",
						got.HTTPStatus,
						tt.wantStatus,
					)
				}
				if followed {
					t.Fatal("followed redirect; want first response only")
				}
				return
			}

			tr = &seqTransport{
				fn: func(req *http.Request, _ int) (*http.Response, error) {
					var body io.ReadCloser = io.NopCloser(
						strings.NewReader(tt.body),
					)
					if tt.bodyErr != nil {
						body = io.NopCloser(&errReadCloser{err: tt.bodyErr})
					}
					return &http.Response{
						StatusCode: tt.status,
						Body:       body,
						Header:     make(http.Header),
						Request:    req,
					}, nil
				},
			}
			site := catalog.Site{
				Name:       tt.name,
				HomeURL:    "https://example.com",
				ProfileURL: "https://example.com/{username}",
				Method:     tt.method,
				Check:      tt.check,
			}
			sc, err := scan.New(scan.Options{
				Client:  &http.Client{Transport: tr},
				Workers: 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			got := sc.Check(context.Background(), "alice", site)
			if got.Exists != tt.want {
				t.Fatalf(
					"exists = %v, want %v (err=%v)",
					got.Exists,
					tt.want,
					got.Err,
				)
			}
			if tt.want != scan.ErrorState && got.HTTPStatus != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					got.HTTPStatus,
					tt.wantStatus,
				)
			}
			if tt.want == scan.ErrorState && got.Err == nil {
				t.Fatal("want body read error")
			}
		})
	}
}

type errReadCloser struct{ err error }

func (e *errReadCloser) Read([]byte) (int, error) { return 0, e.err }
func (e *errReadCloser) Close() error             { return nil }

func TestRunCancelMidFlight(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-block:
			case <-r.Context().Done():
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(block) })

	sites := make([]catalog.Site, 40)
	for i := range sites {
		sites[i] = catalog.Site{
			Name:       "Site" + strconv.Itoa(i),
			HomeURL:    srv.URL,
			ProfileURL: srv.URL + "/{username}/" + strconv.Itoa(i),
			Method:     http.MethodGet,
			Check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
		}
	}

	sc, err := scan.New(scan.Options{
		Client:  srv.Client(),
		Workers: 8,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	out := sc.Run(ctx, "alice", sites)
	cancel()

	n := 0
	for range out {
		n++
	}
	// Channel must close; some results may have been sent before cancel.
	t.Logf("received %d results before cancel drained", n)
}

func TestStatusCheckSkipsMultiMiBBody(t *testing.T) {
	var readBytes int
	payload := strings.Repeat("y", 2<<20)
	tr := &seqTransport{
		fn: func(req *http.Request, _ int) (*http.Response, error) {
			body := &countingReadCloser{
				r: io.NopCloser(strings.NewReader(payload)),
				n: &readBytes,
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	}
	sites := make([]catalog.Site, 5)
	for i := range sites {
		sites[i] = catalog.Site{
			Name:       "Big" + strconv.Itoa(i),
			HomeURL:    "https://example.com",
			ProfileURL: "https://example.com/{username}/" + strconv.Itoa(i),
			Method:     http.MethodGet,
			Check: catalog.Check{
				Type:           catalog.CheckStatus,
				NotFoundStatus: []int{404},
			},
		}
	}
	sc, err := scan.New(scan.Options{
		Client:  &http.Client{Transport: tr},
		Workers: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	for range sc.Run(context.Background(), "alice", sites) {
	}
	if readBytes != 0 {
		t.Fatalf("body bytes read = %d, want 0 across catalog", readBytes)
	}
}
