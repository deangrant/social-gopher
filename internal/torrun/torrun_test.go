package torrun_test

import (
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/deangrant/social-gopher/internal/torrun"
)

func startFakeSOCKS5(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(
		context.Background(),
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handleFakeSOCKS5(c)
		}
	}()
	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

func handleFakeSOCKS5(c net.Conn) {
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(time.Second))
	var greet [3]byte
	if _, err := io.ReadFull(c, greet[:]); err != nil {
		return
	}
	if greet[0] != 0x05 {
		return
	}
	_, _ = c.Write([]byte{0x05, 0x00})
}

func TestEnsureReusesExistingListener(t *testing.T) {
	addr, stop := startFakeSOCKS5(t)
	defer stop()

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
	addr, stop := startFakeSOCKS5(t)
	defer stop()

	inst, err := torrun.Ensure(context.Background(), torrun.Options{
		SocksAddr: addr,
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

func TestEnsureRejectsNonSocksListener(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(
		context.Background(),
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = torrun.Ensure(ctx, torrun.Options{
		SocksAddr:      ln.Addr().String(),
		Binary:         filepath.Join(t.TempDir(), "missing-tor"),
		StartupTimeout: time.Second,
	})
	if err == nil {
		t.Fatal("Ensure() error = nil, want non-SOCKS error")
	}
	if !strings.Contains(err.Error(), "is not a SOCKS5 proxy") {
		t.Fatalf("Ensure() error = %v, want not SOCKS5", err)
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

func writeTorStub(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "stub.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "tor-stub")
	cmd := exec.Command("go", "build", "-o", bin, srcPath)
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build stub: %v\n%s", err, out)
	}
	return bin
}

func TestEnsureStartupTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	stub := writeTorStub(t, `package main
import "time"
func main() { time.Sleep(30 * time.Second) }
`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = torrun.Ensure(ctx, torrun.Options{
		SocksAddr:      addr,
		Binary:         stub,
		StartupTimeout: 300 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Ensure() error = nil, want startup timeout")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Ensure() error = %v, want timed out", err)
	}
}

func TestEnsureCloseKillsStub(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	stub := writeTorStub(t, `package main
import (
	"io"
	"net"
	"os"
	"os/signal"
	"time"
)
func main() {
	signal.Ignore(os.Interrupt)
	var addr string
	for i, a := range os.Args {
		if a == "--SocksPort" && i+1 < len(os.Args) {
			addr = os.Args[i+1]
		}
	}
	if addr == "" {
		os.Exit(2)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		os.Exit(1)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(time.Second))
				buf := make([]byte, 3)
				_, _ = io.ReadFull(c, buf)
				_, _ = c.Write([]byte{0x05, 0x00})
			}(c)
		}
	}()
	select {}
}
`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inst, err := torrun.Ensure(ctx, torrun.Options{
		SocksAddr:      addr,
		Binary:         stub,
		StartupTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if !inst.StartedByUs() {
		t.Fatal("StartedByUs = false, want true")
	}
	pid := inst.Pid()
	if pid == 0 {
		t.Fatal("Pid = 0")
	}
	if err := inst.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if err != nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid %d still alive after Close", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
