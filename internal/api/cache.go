package api

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Keep authentication and MCP credentials out of caches, including error responses.
func sensitiveResponseNoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/auth/") || strings.HasPrefix(c.Request.URL.Path, "/api/mcp/") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}
