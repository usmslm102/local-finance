package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLocalAccessNavigationAndLegacyMutations(t *testing.T) {
	router := gin.New()
	router.Use(localRequestProtection())
	router.Any("/", func(c *gin.Context) { c.Status(204) })
	router.Any("/api/action", func(c *gin.Context) { c.Status(204) })
	for _, tc := range []struct {
		name, method, path string
		headers            map[string]string
		want               int
	}{
		{"legacy foreign API read", "GET", "/api/action", map[string]string{"Referer": "https://example.com/docs"}, 403},
		{"metadata-free API read", "GET", "/api/action", nil, 403},
		{"explicit API reader", "GET", "/api/action", map[string]string{"X-LocalFinance-Request": "1"}, 204},
		{"legacy local API read", "GET", "/api/action", map[string]string{"Referer": "http://127.0.0.1:8080/transactions"}, 204},
		{"external document link", "GET", "/", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document", "Referer": "https://example.com/docs"}, 204},
		{"external API navigation", "GET", "/api/action", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}, 403},
		{"external iframe", "GET", "/", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "iframe"}, 403},
		{"legacy foreign form", "POST", "/api/action", map[string]string{"Referer": "https://example.com/form", "Content-Type": "application/x-www-form-urlencoded"}, 403},
		{"anonymous form", "POST", "/api/action", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, 403},
		{"anonymous delete", "DELETE", "/api/action", nil, 403},
		{"anonymous JSON", "POST", "/api/action", map[string]string{"Content-Type": "application/json"}, 403},
		{"legacy local form", "POST", "/api/action", map[string]string{"Referer": "http://127.0.0.1:8080/import?tab=bank"}, 204},
		{"legacy development form", "POST", "/api/action", map[string]string{"Referer": "http://localhost:5173/import"}, 204},
		{"explicit local client", "DELETE", "/api/action", map[string]string{"X-LocalFinance-Request": "1"}, 204},
		{"foreign origin with explicit header", "POST", "/api/action", map[string]string{"Origin": "https://example.com", "X-LocalFinance-Request": "1"}, 403},
		{"foreign referrer with explicit header", "POST", "/api/action", map[string]string{"Referer": "https://example.com", "X-LocalFinance-Request": "1"}, 403},
		{"cross-site mutation", "POST", "/api/action", map[string]string{"Sec-Fetch-Site": "cross-site", "X-LocalFinance-Request": "1"}, 403},
		{"same-origin browser", "PUT", "/api/action", map[string]string{"Sec-Fetch-Site": "same-origin"}, 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://127.0.0.1:8080"+tc.path, strings.NewReader("{}"))
			for name, value := range tc.headers {
				req.Header.Set(name, value)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d", w.Code, tc.want)
			}
		})
	}
}
