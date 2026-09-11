package ginweb

import (
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/constants"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/middlewares/bodylimit"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/middlewares/compress"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/middlewares/log"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/middlewares/problemdetail"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/middlewares/requestid"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/logger"
)

func AddGin(container *dig.Container) error {
	err := container.Provide(func(l logger.Logger) *gin.Engine {
		gin.SetMode(gin.ReleaseMode)

		e := gin.New()

		// The router answers the URL exactly as it was requested: no trailing
		// slash is added or removed, no doubled separator is collapsed, and a
		// percent-encoded separator stays encoded inside a path parameter.
		e.RedirectTrailingSlash = false
		e.RedirectFixedPath = false
		e.UseRawPath = true
		e.UnescapePathValues = false
		e.HandleMethodNotAllowed = true

		// Diagnostic and documentation endpoints are excluded from the request
		// pipeline so that they neither pollute the access log nor have their
		// payloads rewritten on the way out.
		skipper := func(c *gin.Context) bool {
			path := c.Request.URL.Path

			return strings.Contains(path, "swagger") ||
				strings.Contains(path, "metrics") ||
				strings.Contains(path, "health") ||
				strings.Contains(path, "favicon.ico")
		}

		e.Use(log.GinLogger(l, log.WithSkipper(skipper)))
		// Outermost renderer, so a failure raised by the body-size guard or by
		// any other middleware still leaves the client with the service's error
		// envelope instead of an empty response. It writes only when nothing
		// else has, which keeps the inner renderer below authoritative for
		// failures raised by the endpoints themselves.
		e.Use(problemdetail.ProblemDetail(l, problemdetail.WithSkipper(skipper)))
		e.Use(bodylimit.BodyLimit(constants.BodyLimit))
		e.Use(requestid.RequestID())
		e.Use(compress.Gzip(constants.GzipLevel, skipper))
		e.Use(problemdetail.ProblemDetail(l, problemdetail.WithSkipper(skipper)))

		e.NoRoute(func(c *gin.Context) {
			// Skipped paths get no error body at all, so an unmapped
			// diagnostic endpoint answers empty rather than with the
			// service's error envelope.
			if skipper(c) {
				c.Status(http.StatusOK)

				return
			}

			_ = c.Error(webcore.ErrNotFound)
		})

		e.NoMethod(func(c *gin.Context) {
			if allow := c.Writer.Header().Get("Allow"); allow != "" {
				c.Writer.Header().Set("Allow", AllowHeader(strings.Split(allow, ",")))
			}

			if skipper(c) {
				c.Status(http.StatusOK)

				return
			}

			_ = c.Error(webcore.ErrMethodNotAllowed)
		})

		return e
	})

	return err
}
