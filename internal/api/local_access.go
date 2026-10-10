package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

var localDevOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:3000", "http://127.0.0.1:3000"}

// Validate the browser boundary before serving either the SPA or API.
func localRequestProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		hostname := host
		if h, _, err := net.SplitHostPort(host); err == nil {
			hostname = h
		}
		if hostname != "127.0.0.1" && hostname != "localhost" && hostname != "::1" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Local requests only"})
			return
		}
		origin := c.GetHeader("Origin")
		if origin != "" {
			if !trustedLocalURL(origin, host, false) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid request origin"})
				return
			}
		}
		safe := c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions
		apiRequest := c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/")
		// External links may open the SPA, but never expose API responses or
		// permit iframe embedding or cross-site mutations.
		documentNavigation := (c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead) &&
			c.GetHeader("Sec-Fetch-Mode") == "navigate" && c.GetHeader("Sec-Fetch-Dest") == "document" &&
			!apiRequest
		if c.GetHeader("Sec-Fetch-Site") == "cross-site" && !documentNavigation {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if !safe || apiRequest && c.Request.Method != http.MethodOptions {
			referer := c.GetHeader("Referer")
			if referer != "" && !trustedLocalURL(referer, host, true) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			// Legacy browsers may omit Origin and Fetch Metadata. A custom
			// header cannot be sent by a cross-site HTML form; native clients
			// can explicitly supply it. It is a CSRF boundary, not authentication.
			if origin == "" && referer == "" && c.GetHeader("Sec-Fetch-Site") != "same-origin" && c.GetHeader("X-LocalFinance-Request") != "1" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Local request origin or X-LocalFinance-Request: 1 required"})
				return
			}
		}
		c.Next()
	}
}

func trustedLocalURL(value, host string, referer bool) bool {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return false
	}
	if !referer && (u.Path != "" || u.RawQuery != "" || u.ForceQuery) {
		return false
	}
	return u.Host == host || isLocalDevOrigin(u.Scheme+"://"+u.Host)
}

func isLocalDevOrigin(origin string) bool {
	for _, allowed := range localDevOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
