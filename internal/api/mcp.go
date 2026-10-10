package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetMCPSettings(c *gin.Context) { c.JSON(http.StatusOK, h.mcpManager.Status()) }
func (h *Handler) UpdateMCPSettings(c *gin.Context) {
	var request struct {
		AllowStatementWrites      *bool `json:"allow_statement_writes"`
		Enabled                   *bool `json:"enabled"`
		Port                      *int  `json:"port"`
		AllowCategorizationWrites *bool `json:"allow_categorization_writes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Enabled == nil || request.Port == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide enabled and port"})
		return
	}
	status, err := h.mcpManager.ConfigureAccess(*request.Enabled, *request.Port, request.AllowCategorizationWrites, request.AllowStatementWrites)
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
