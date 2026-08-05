// Package transport builds HTTP clients for site probes, including Tor/proxy.
package transport

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// DefaultTorProxy is the local Tor SOCKS5 endpoint with remote DNS.
const DefaultTorProxy = "socks5h://127.0.0.1:9050"

// Options configures an HTTP client used for probing.
type Options struct {
	Timeout   time.Duration
	ProxyURL  string
	UseTor    bool
	UserAgent string
}

// Result holds the client and any notice about proxy selection.
type Result struct {
	Client *http.Client
	Notice string
}

// New builds an HTTP client from opts.
// If both UseTor and ProxyURL are set, ProxyURL wins and Notice explains that.
func New(opts Options) (Result, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 20 * time.Second
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "social-gopher/1.0 (+https://github.com/deangrant/social-gopher)"
	}

	proxyURL, notice := resolveProxy(opts)
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   opts.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return Result{}, fmt.Errorf("parse proxy URL: %w", err)
		}
		switch u.Scheme {
		case "socks5", "socks5h":
			var auth *proxy.Auth
			if u.User != nil {
				pass, _ := u.User.Password()
				auth = &proxy.Auth{
					User:     u.User.Username(),
					Password: pass,
				}
			}
			dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
			if err != nil {
				return Result{}, fmt.Errorf("socks5 dialer: %w", err)
			}
			if cd, ok := dialer.(proxy.ContextDialer); ok {
				transport.DialContext = cd.DialContext
			} else {
				transport.DialContext = nil
				transport.Dial = dialer.Dial
			}
			transport.Proxy = nil
		case "http", "https":
			transport.Proxy = http.ProxyURL(u)
		default:
			return Result{}, fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
		}
	}

	client := &http.Client{
		Timeout:   opts.Timeout,
		Transport: &uaTransport{base: transport, userAgent: opts.UserAgent},
	}
	return Result{Client: client, Notice: notice}, nil
}

func resolveProxy(opts Options) (proxyURL, notice string) {
	if opts.ProxyURL != "" {
		if opts.UseTor {
			notice = "both --tor and --proxy set; using --proxy"
		}
		return opts.ProxyURL, notice
	}
	if opts.UseTor {
		return DefaultTorProxy, ""
	}
	return "", ""
}

// uaTransport injects a User-Agent when the request has none.
type uaTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(req)
}
