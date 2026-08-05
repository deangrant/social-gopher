package transport_test

import (
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/deangrant/social-gopher/internal/transport"
)

func TestNewDirect(t *testing.T) {
	res, err := transport.New(transport.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if res.Client == nil {
		t.Fatal("Client is nil")
	}
	if res.Notice != "" {
		t.Fatalf("Notice = %q, want empty", res.Notice)
	}
}

func TestDirectIgnoresEnvProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:1")

	res, err := transport.New(transport.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if transport.ClientProxyForTest(res.Client) != nil {
		t.Fatal("direct mode must not use environment proxy")
	}
}

func TestProxyWinsOverTor(t *testing.T) {
	res, err := transport.New(transport.Options{
		Timeout:  time.Second,
		UseTor:   true,
		ProxyURL: "socks5h://127.0.0.1:9150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Notice == "" {
		t.Fatal("want notice when both tor and proxy set")
	}
	if res.Client == nil {
		t.Fatal("Client is nil")
	}
}

func TestTorDefault(t *testing.T) {
	res, err := transport.New(transport.Options{
		Timeout: time.Second,
		UseTor:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Client == nil {
		t.Fatal("Client is nil")
	}
	if res.Notice != "" {
		t.Fatalf("Notice = %q, want empty", res.Notice)
	}
}

func TestUnsupportedScheme(t *testing.T) {
	_, err := transport.New(transport.Options{
		ProxyURL: "ftp://127.0.0.1:21",
	})
	if err == nil {
		t.Fatal("want error for unsupported scheme")
	}
}

func TestSocks5RemoteDNSNotice(t *testing.T) {
	res, err := transport.New(transport.Options{
		Timeout:  time.Second,
		ProxyURL: "socks5://127.0.0.1:9050",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Client == nil {
		t.Fatal("Client is nil")
	}
	want := "socks5:// uses remote DNS (same as socks5h); unlike curl"
	if res.Notice != want {
		t.Fatalf("Notice = %q, want %q", res.Notice, want)
	}
}

func TestSocks5hNoDNSNotice(t *testing.T) {
	res, err := transport.New(transport.Options{
		Timeout:  time.Second,
		ProxyURL: "socks5h://127.0.0.1:9050",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Notice != "" {
		t.Fatalf("Notice = %q, want empty for socks5h", res.Notice)
	}
}

func TestSocks5hSendsHostname(t *testing.T) {
	var (
		mu   sync.Mutex
		atyp byte
		host string
	)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(2 * time.Second))
		var greet [3]byte
		if _, err := io.ReadFull(c, greet[:]); err != nil {
			return
		}
		_, _ = c.Write([]byte{0x05, 0x00})
		hdr := make([]byte, 4)
		if _, err := io.ReadFull(c, hdr); err != nil {
			return
		}
		mu.Lock()
		atyp = hdr[3]
		mu.Unlock()
		switch hdr[3] {
		case 0x01: // IPv4
			addr := make([]byte, 4+2)
			_, _ = io.ReadFull(c, addr)
			mu.Lock()
			host = net.IP(addr[:4]).String()
			mu.Unlock()
		case 0x03: // domain
			var n [1]byte
			if _, err := io.ReadFull(c, n[:]); err != nil {
				return
			}
			name := make([]byte, int(n[0])+2)
			if _, err := io.ReadFull(c, name); err != nil {
				return
			}
			mu.Lock()
			host = string(name[:n[0]])
			mu.Unlock()
		case 0x04: // IPv6
			addr := make([]byte, 16+2)
			_, _ = io.ReadFull(c, addr)
			mu.Lock()
			host = net.IP(addr[:16]).String()
			mu.Unlock()
		}
		// General failure reply so the client stops.
		_, _ = c.Write([]byte{
			0x05, 0x01, 0x00, 0x01,
			0, 0, 0, 0,
			0, 0,
		})
	}()

	proxyAddr := ln.Addr().String()
	res, err := transport.New(transport.Options{
		Timeout:  2 * time.Second,
		ProxyURL: "socks5h://" + proxyAddr,
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"http://probe.example/{username}",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = res.Client.Do(req)
	<-done

	mu.Lock()
	defer mu.Unlock()
	if atyp != 0x03 {
		t.Fatalf("SOCKS ATYP = %#x, want 0x03 (domain); host=%q", atyp, host)
	}
	if host != "probe.example" {
		t.Fatalf("SOCKS host = %q, want probe.example", host)
	}
}
