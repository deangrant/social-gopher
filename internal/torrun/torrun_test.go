package torrun_test

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/deangrant/social-gopher/internal/torrun"
)

func TestEnsureReusesExistingListener(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(
		context.Background(),
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	inst, err := torrun.Ensure(ctx, torrun.Options{
		SocksAddr:      addr,
		Binary:         filepath.Join(t.TempDir(), "missing-tor"),
		StartupTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	defer inst.Close()

	if inst.StartedByUs() {
		t.Fatal("StartedByUs = true, want false when SOCKS already listening")
	}
	if got := inst.ProxyURL(); got != "socks5h://"+addr {
		t.Fatalf("ProxyURL = %q, want socks5h://%s", got, addr)
	}
	if inst.Pid() != 0 {
		t.Fatalf("Pid = %d, want 0", inst.Pid())
	}
}

func TestCloseNoopWhenNotStartedByUs(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(
		context.Background(),
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	inst, err := torrun.Ensure(context.Background(), torrun.Options{
		SocksAddr: ln.Addr().String(),
	})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if err := inst.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := inst.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestEnsureMissingBinary(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(
		context.Background(),
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = torrun.Ensure(ctx, torrun.Options{
		SocksAddr:      addr,
		Binary:         filepath.Join(t.TempDir(), "no-such-tor"),
		StartupTimeout: 500 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Ensure() error = nil, want error when binary missing")
	}
}
