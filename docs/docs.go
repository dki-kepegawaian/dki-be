// Package docs menyajikan dokumentasi API: spesifikasi OpenAPI di /docs/openapi.yaml
// dan tampilan Scalar di /docs. Keduanya di-embed ke binary.
package docs

import (
	_ "embed"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var spec []byte

// Versi Scalar dipin supaya tampilan tidak berubah tanpa sengaja.
const scalarHTML = `<!doctype html>
<html lang="id">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Teralur API</title>
</head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.76.2/dist/browser/standalone.js"></script>
  <script>
    Scalar.createApiReference('#app', {
      url: '/docs/openapi.yaml',
      persistAuth: true,
    })
  </script>
</body>
</html>`

// RegisterRoutes memasang /docs. Di APP_ENV=production docs dimatikan,
// kecuali DOCS_ENABLED=true.
func RegisterRoutes(server *gin.Engine) {
	if os.Getenv("APP_ENV") == "production" && os.Getenv("DOCS_ENABLED") != "true" {
		return
	}
	server.GET("/docs", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarHTML))
	})
	server.GET("/docs/openapi.yaml", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", spec)
	})
}
