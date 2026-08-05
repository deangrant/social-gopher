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
