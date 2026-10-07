package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetMCPSettings(c *gin.Context) { c.JSON(http.StatusOK, h.mcpManager.Status()) }
func (h *Handler) UpdateMCPSettings(c *gin.Context) {
	var request struct {
		Enabled *bool `json:"enabled"`
		Port    *int  `json:"port"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Enabled == nil || request.Port == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide enabled and port"})
		return
	}
	status, err := h.mcpManager.Configure(*request.Enabled, *request.Port)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}
func (h *Handler) RotateMCPToken(c *gin.Context) {
	token, err := h.mcpManager.RotateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create MCP token"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"token": token, "settings": h.mcpManager.Status()})
}

// MCP management is local-only and cannot be driven by a cross-origin website.
func mcpManagementProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/mcp/") {
			c.Next()
			return
		}
		c.Header("Cache-Control", "no-store")
		host := c.Request.Host
		hostname := host
		if h, _, err := net.SplitHostPort(host); err == nil {
			hostname = h
		}
		if hostname != "127.0.0.1" && hostname != "localhost" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Local MCP management only"})
			return
		}
		if origin := c.GetHeader("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Scheme != "http" || (u.Host != host && !isMCPDevOrigin(origin)) || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid management origin"})
				return
			}
		}
		if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func isMCPDevOrigin(origin string) bool {
	switch origin {
	case "http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:3000", "http://127.0.0.1:3000":
		return true
	default:
		return false
	}
}
