package transport

import (
	"net/http"
	"net/url"
)

// ClientProxyForTest returns the underlying http.Transport Proxy func.
func ClientProxyForTest(c *http.Client) func(*http.Request) (*url.URL, error) {
	ut := c.Transport.(*uaTransport)
	tr := ut.base.(*http.Transport)
	return tr.Proxy
}
