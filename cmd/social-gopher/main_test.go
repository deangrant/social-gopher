package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExitOnInterrupt(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: 0},
		{name: "canceled", err: context.Canceled, want: 130},
		{
			name: "wrapped canceled",
			err:  errors.Join(context.Canceled),
			want: 130,
		},
		{name: "deadline", err: context.DeadlineExceeded, want: 1},
		{name: "other", err: errors.New("boom"), want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exitOnInterrupt(tt.err)
			if got != tt.want {
				t.Fatalf(
					"exitOnInterrupt(%v) = %d, want %d",
					tt.err,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestCSVOutputPath(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		want    string
		wantErr bool
	}{
		{name: "simple", user: "alice", want: "alice.csv"},
		{name: "slash", user: "a/b", want: "a_b.csv"},
		{name: "backslash", user: `a\b`, want: "a_b.csv"},
		{name: "dotdot_slash", user: "../x", want: ".._x.csv"},
		{name: "nul", user: "a\x00b", want: "a_b.csv"},
		{name: "dotdot", user: "..", wantErr: true},
		{name: "dot", user: ".", wantErr: true},
		{name: "empty", user: "", wantErr: true},
		{name: "only_sep", user: "/", want: "_.csv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := csvOutputPath(tt.user)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("csvOutputPath(%q) = %q, want error", tt.user, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("csvOutputPath(%q) error = %v", tt.user, err)
			}
			if got != tt.want {
				t.Fatalf(
					"csvOutputPath(%q) = %q, want %q",
					tt.user,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestRunRejectsNonPositiveWorkers(t *testing.T) {
	for _, args := range [][]string{
		{"-workers", "0", "alice"},
		{"-workers", "-1", "alice"},
		{"-workers", "0", "-validate-catalog"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if code := run(args); code != 2 {
				t.Fatalf("run(%v) = %d, want 2", args, code)
			}
		})
	}
}

func TestRunRejectsEmptyUsername(t *testing.T) {
	cat := writeTempCatalog(t)
	for _, user := range []string{"", "   "} {
		t.Run(user, func(t *testing.T) {
			code := run([]string{"-catalog", cat, user})
			if code != 2 {
				t.Fatalf("run(username %q) = %d, want 2", user, code)
			}
		})
	}
}

func writeTempCatalog(t *testing.T) string {
	t.Helper()
	const src = `{
		"sites":[{
			"name":"Example",
			"home_url":"https://example.com",
			"profile_url":"https://example.com/{username}",
			"check":{"type":"status"},
			"profile":"default"
		}]
	}`
	path := filepath.Join(t.TempDir(), "sites.json")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunValidateCatalogArgs(t *testing.T) {
	cat := writeTempCatalog(t)

	if code := run([]string{
		"-validate-catalog",
		"-catalog", cat,
		"alice",
	}); code != 2 {
		t.Fatalf("validate with username = %d, want 2", code)
	}

	if code := run([]string{
		"-validate-catalog",
		"-catalog", cat,
		"-timeout", "1s",
	}); code != 0 {
		t.Fatalf("validate without username = %d, want 0", code)
	}
}

func TestRunCSVUnsafeUsername(t *testing.T) {
	cat := writeTempCatalog(t)
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	code := run([]string{
		"-csv",
		"-catalog", cat,
		"-timeout", "2s",
		"-workers", "1",
		"../x/y",
	})
	if code != 0 {
		t.Fatalf("run(-csv) = %d, want 0", code)
	}
	want := filepath.Join(dir, ".._x_y.csv")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("csv file %s: %v", want, err)
	}
}
