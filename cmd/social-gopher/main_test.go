package main

import (
	"context"
	"errors"
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
