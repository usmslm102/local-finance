package api

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// DesktopActions exposes user-mediated OS actions protected by local origin checks.
// Saving files also requires UI authentication; opening web links works while locked.
// It deliberately exposes neither arbitrary file paths nor shell commands.
type DesktopActions interface {
	SaveFile(name string, content io.Reader) error
	OpenURL(url string) error
}

func registerDesktopRoutes(routes *gin.RouterGroup, desktop DesktopActions) {
	routes.GET("/desktop", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"enabled": desktop != nil}) })
	if desktop == nil {
		return
	}
	routes.POST("/desktop/save", func(c *gin.Context) {
		name := c.Query("name")
		if name == "" || name == "." || name != filepath.Base(name) || strings.ContainsAny(name, "/\\\x00\r\n") || len(name) > 255 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid download filename"})
			return
		}
		if err := desktop.SaveFile(name, c.Request.Body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
	routes.POST("/desktop/open-url", func(c *gin.Context) {
		var body struct {
			URL string `json:"url"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		parsed, err := url.Parse(body.URL)
		if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only HTTP and HTTPS links can be opened"})
			return
		}
		if err := desktop.OpenURL(body.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
