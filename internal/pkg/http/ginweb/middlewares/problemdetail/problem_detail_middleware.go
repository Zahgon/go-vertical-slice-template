package problemdetail

import (
	"github.com/gin-gonic/gin"

	handlers "github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/hadnlers"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/httperrors/problemdetails"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/logger"
)

// ProblemDetail turns whatever error a handler reported into the service's
// problem-detail response body. It is the outermost middleware, so it is the
// last thing to run on the way out and therefore sees errors raised anywhere
// below it, including by the routing fallbacks.
func ProblemDetail(l logger.Logger, opts ...Option) gin.HandlerFunc {
	cfg := &config{Skipper: webcore.DefaultSkipper}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	if cfg.Skipper == nil {
		cfg.Skipper = webcore.DefaultSkipper
	}

	return func(c *gin.Context) {
		if cfg.Skipper(c) {
			c.Next()

			return
		}

		c.Next()

		ginErr := c.Errors.Last()
		if ginErr == nil {
			return
		}

		err := ginErr.Err

		var prbError problemDetails.ProblemDetailErr
		if cfg.ProblemParser != nil {
			prbError = cfg.ProblemParser(err)
		} else {
			prbError = problemDetails.ParseError(err)
		}

		if prbError != nil {
			handlers.ProblemDetailErrorHandlerFunc(prbError, c, l)
		}
	}
}
