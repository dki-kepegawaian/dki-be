package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BatasiBody membatasi ukuran body request: maxMultipart untuk multipart/form-data,
// maxLain untuk selainnya. Body yang melebihi batas gagal dibaca oleh binding.
func BatasiBody(maxLain, maxMultipart int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := maxLain
		if strings.HasPrefix(c.ContentType(), "multipart/") {
			limit = maxMultipart
		}
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"status": false, "message": "request terlalu besar", "error": "ukuran body melebihi batas",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
