// Package testutil contains shared fixtures for local HTTP integration tests.
package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// LocalRequest models an explicit loopback API client. Tests of absent or hostile
// browser headers should construct their own requests instead.
func LocalRequest(method, target string, body io.Reader) *http.Request {
	if strings.HasPrefix(target, "/") {
		target = "http://127.0.0.1:8080" + target
	}
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("X-LocalFinance-Request", "1")
	return req
}
