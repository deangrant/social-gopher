package transport_test

import (
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
