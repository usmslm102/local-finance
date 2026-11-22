package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"local-finance/internal/retirement"
)

func (h *Handler) FIREBaseline(c *gin.Context) {
	result, err := h.db.GetFIREBaseline(time.Now())
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to load retirement planning inputs"})
		return
	}
	c.JSON(200, result)
}
func (h *Handler) FIREProjection(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var input retirement.Scenario
	if err := decoder.Decode(&input); err != nil {
		c.JSON(400, gin.H{"error": "Enter valid retirement inputs using the supported fields"})
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		c.JSON(400, gin.H{"error": "Provide exactly one scenario"})
		return
	}
	result, err := retirement.ProjectContext(c.Request.Context(), input)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}
