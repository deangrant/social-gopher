package scan_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
