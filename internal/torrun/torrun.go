// Package torrun manages a local Tor SOCKS process for -tor scans.
package torrun

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"
)

// DefaultSocksAddr is the usual local Tor SOCKS listener.
const DefaultSocksAddr = "127.0.0.1:9050"

const defaultStartupTimeout = 60 * time.Second

// Options configures Ensure.
type Options struct {
	// SocksAddr is host:port for the SOCKS listener (default 127.0.0.1:9050).
	SocksAddr string
	// Binary is the tor executable path. Empty means LookPath("tor").
	Binary string
	// StartupTimeout bounds how long to wait for SOCKS after starting tor.
	StartupTimeout time.Duration
}

// Instance is a ready Tor SOCKS endpoint. Close stops a process we started.
type Instance struct {
	addr        string
	cmd         *exec.Cmd
	dataDir     string
	startedByUs bool
	waitCh      <-chan error
}

// ProxyURL returns the socks5h URL for this instance.
func (i *Instance) ProxyURL() string {
	return "socks5h://" + i.addr
}

// StartedByUs reports whether Ensure spawned the Tor process.
func (i *Instance) StartedByUs() bool {
	return i.startedByUs
}

// Pid returns the child process id when we started Tor; otherwise 0.
func (i *Instance) Pid() int {
	if i == nil || i.cmd == nil || i.cmd.Process == nil {
		return 0
	}
	return i.cmd.Process.Pid
}

// Close stops the Tor child if we started it and removes its data directory.
func (i *Instance) Close() error {
	if i == nil || !i.startedByUs {
		return nil
	}
	i.startedByUs = false
	if i.cmd != nil && i.cmd.Process != nil {
		_ = i.cmd.Process.Signal(os.Interrupt)
		select {
		case <-i.waitCh:
		case <-time.After(3 * time.Second):
			_ = i.cmd.Process.Kill()
			<-i.waitCh
		}
	}
	if i.dataDir != "" {
		_ = os.RemoveAll(i.dataDir)
		i.dataDir = ""
	}
	return nil
}

// Ensure makes a local Tor SOCKS endpoint available.
// If SocksAddr already speaks SOCKS5, it returns a no-op Instance.
// If something else is listening there, it returns an error.
// Otherwise it starts tor and waits until SOCKS is ready or ctx is done.
func Ensure(ctx context.Context, opts Options) (*Instance, error) {
	if opts.SocksAddr == "" {
		opts.SocksAddr = DefaultSocksAddr
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = defaultStartupTimeout
	}

	switch probeSocks5(ctx, opts.SocksAddr) {
	case socksProbeOK:
		return &Instance{addr: opts.SocksAddr}, nil
	case socksProbeNotSocks:
		return nil, fmt.Errorf(
			"%s is listening but is not a SOCKS5 proxy",
			opts.SocksAddr,
		)
	}

	binary := opts.Binary
	if binary == "" {
		path, err := exec.LookPath("tor")
		if err != nil {
			return nil, fmt.Errorf(
				"tor not found on PATH; install Tor "+
					"(e.g. apt install tor) or start it so %s is listening",
				opts.SocksAddr,
			)
		}
		binary = path
	}

	dataDir, err := os.MkdirTemp("", "social-gopher-tor-*")
	if err != nil {
		return nil, fmt.Errorf("tor data directory: %w", err)
	}

	// Process lifetime is owned by Instance.Close, not Ensure's ctx.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), binary,
		"--SocksPort", opts.SocksAddr,
		"--DataDirectory", dataDir,
		"--IgnoreMissingTorrc", "1",
		"--ClientOnly", "1",
	)
	cmd.Stdout = io.Discard
	var stderr bufLimited
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("start tor: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	deadline := time.Now().Add(opts.StartupTimeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = killCmd(cmd)
			<-waitCh
			_ = os.RemoveAll(dataDir)
			return nil, fmt.Errorf("waiting for Tor: %w", ctx.Err())
		case err := <-waitCh:
			_ = os.RemoveAll(dataDir)
			detail := stderr.String()
			if detail != "" {
				return nil, fmt.Errorf(
					"tor exited before SOCKS was ready: %w (%s)",
					err,
					detail,
				)
			}
			return nil, fmt.Errorf("tor exited before SOCKS was ready: %w", err)
		case <-ticker.C:
			switch probeSocks5(ctx, opts.SocksAddr) {
			case socksProbeOK:
				return &Instance{
					addr:        opts.SocksAddr,
					cmd:         cmd,
					dataDir:     dataDir,
					startedByUs: true,
					waitCh:      waitCh,
				}, nil
			case socksProbeNotSocks:
				_ = killCmd(cmd)
				<-waitCh
				_ = os.RemoveAll(dataDir)
				return nil, fmt.Errorf(
					"%s is listening but is not a SOCKS5 proxy",
					opts.SocksAddr,
				)
			}
			if time.Now().After(deadline) {
				_ = killCmd(cmd)
				<-waitCh
				_ = os.RemoveAll(dataDir)
				return nil, fmt.Errorf(
					"timed out waiting for Tor SOCKS on %s",
					opts.SocksAddr,
				)
			}
		}
	}
}

type socksProbeResult int

const (
	socksProbeClosed socksProbeResult = iota
	socksProbeOK
	socksProbeNotSocks
)

// probeSocks5 dials addr and checks a SOCKS5 no-auth greeting reply.
func probeSocks5(ctx context.Context, addr string) socksProbeResult {
	d := net.Dialer{Timeout: 200 * time.Millisecond}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return socksProbeClosed
	}
	defer c.Close()

	_ = c.SetDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := c.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		return socksProbeNotSocks
	}
	var reply [2]byte
	if _, err := io.ReadFull(c, reply[:]); err != nil {
		return socksProbeNotSocks
	}
	if reply[0] != 0x05 || reply[1] != 0x00 {
		return socksProbeNotSocks
	}
	return socksProbeOK
}

func killCmd(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(os.Interrupt)
	time.Sleep(200 * time.Millisecond)
	return cmd.Process.Kill()
}

// bufLimited keeps a small stderr sample for error messages.
type bufLimited struct {
	b []byte
}

func (b *bufLimited) Write(p []byte) (int, error) {
	const limit = 2048
	if len(b.b) < limit {
		need := limit - len(b.b)
		if need > len(p) {
			need = len(p)
		}
		b.b = append(b.b, p[:need]...)
	}
	return len(p), nil
}

func (b *bufLimited) String() string {
	return string(b.b)
}
