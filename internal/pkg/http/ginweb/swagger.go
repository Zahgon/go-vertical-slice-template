package ginweb

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	swaggerFilesFS "github.com/swaggo/files/v2"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
)

// SwaggerHandler serves the Swagger UI mounted under prefix.
//
// The generated index page and the OpenAPI document are rendered by the
// swagger adapter, while every other asset is streamed straight from the
// bundled Swagger UI filesystem so unknown names answer with the standard
// file-server response. Requesting the bare prefix redirects to the index
// page, keeping the redirect target the mount point produces.
func SwaggerHandler(prefix string, param string) gin.HandlerFunc {
	adapter := ginSwagger.WrapHandler(swaggerFiles.Handler)
	assets := http.FileServer(http.FS(swaggerFilesFS.FS))

	return func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param(param), "/")

		switch name {
		case "":
			c.Writer.Header().Set(webcore.HeaderLocation, prefix+"/"+"/index.html")
			c.Status(http.StatusMovedPermanently)
			c.Writer.WriteHeaderNow()
			c.Writer.Flush()
		case "index.html", "doc.json":
			adapter(c)
		default:
			c.Request.URL.Path = name
			assets.ServeHTTP(c.Writer, c.Request)
			c.Writer.Flush()
		}
	}
}
