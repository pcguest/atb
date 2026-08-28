// SPDX-License-Identifier: MIT
package proxy

import "net/http"

// NewProxyForTest builds a proxy with an injected recorder and a session
// manager wired to that recorder's close callback. It is a test-only seam so
// black-box tests in package proxy_test can exercise the capture path without
// reaching into unexported fields.
func NewProxyForTest(cfg ProxyConfig, rec *BundleRecorder) (*Proxy, error) {
	p, err := NewProxy(cfg, LoggingHandler{}, nil)
	if err != nil {
		return nil, err
	}
	p.recorder = rec
	p.sessions = NewSessionManager(rec.sessionCloseCallback)
	return p, nil
}

// CaptureRequestForTest exposes request capture to black-box tests.
func (p *Proxy) CaptureRequestForTest(host string, req *http.Request, body []byte) error {
	f := &forwarder{proxy: p}
	return f.captureRequest(host, req, body)
}

// CaptureResponseForTest exposes response capture to black-box tests.
func (p *Proxy) CaptureResponseForTest(host string, req *http.Request, resp *http.Response, body []byte) error {
	f := &forwarder{proxy: p}
	return f.captureResponse(host, req, resp, body)
}
