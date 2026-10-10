package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"local-finance/internal/db"
	"local-finance/internal/service"
)

const maxBatchFiles = 20
const maxBatchBytes int64 = 64 << 20

// Multipart memory limits only control spilling to disk. Cap the request before
// parsing, and release temporary files on success and every error path.
func requestBodyLimits() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := int64(1 << 20)
		fileLimit := int64(0)
		batch := false
		switch c.Request.URL.Path {
		case "/api/statements/upload", "/api/statements/preview":
			fileLimit = service.MaxStatementFileSize
			limit = fileLimit + (1 << 20)
		case "/api/statements/upload-batch", "/api/statements/preview-batch":
			fileLimit = service.MaxStatementFileSize
			limit, batch = maxBatchBytes+(1<<20), true
		case "/api/database/restore":
			fileLimit = db.MaxRestoreSize
			limit = fileLimit + (1 << 20)
		case "/api/investments/import", "/api/investments/preview":
			fileLimit = service.MaxInvestmentFileSize
			limit = fileLimit + (1 << 20)
		}
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Upload exceeds the allowed size"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		if fileLimit > 0 {
			err := c.Request.ParseMultipartForm(8 << 20)
			if c.Request.MultipartForm != nil {
				defer c.Request.MultipartForm.RemoveAll()
			}
			if err != nil {
				status := http.StatusBadRequest
				var sizeError *http.MaxBytesError
				if errors.As(err, &sizeError) {
					status = http.StatusRequestEntityTooLarge
				}
				c.AbortWithStatusJSON(status, gin.H{"error": "Invalid multipart upload or upload exceeds the allowed size"})
				return
			}
			count, size := 0, int64(0)
			for _, files := range c.Request.MultipartForm.File {
				for _, file := range files {
					count++
					size += file.Size
					if file.Size > fileLimit {
						c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File exceeds the allowed size"})
						return
					}
				}
			}
			if batch && (count > maxBatchFiles || size > maxBatchBytes) || !batch && count > 1 {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Too many files or batch exceeds 64 MB; upload at most 20 statements at a time"})
				return
			}
		}
		c.Next()
	}
}
