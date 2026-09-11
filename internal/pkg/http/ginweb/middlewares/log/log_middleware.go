package log

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/logger"
)

// GinLogger writes one structured access record per request and then a second
// record whose severity follows the response status, so a failing request is
// visible at error level without having to grep successful traffic.
func GinLogger(l logger.Logger, opts ...Option) gin.HandlerFunc {
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

		req := c.Request
		start := time.Now()

		c.Next()

		latency := time.Since(start)

		status := c.Writer.Status()
		size := int64(c.Writer.Size())
		if size < 0 {
			size = 0
		}

		id := req.Header.Get(webcore.HeaderXRequestID)
		if id == "" {
			id = c.Writer.Header().Get(webcore.HeaderXRequestID)
		}

		var requestErr error
		if ginErr := c.Errors.Last(); ginErr != nil {
			requestErr = ginErr.Err
		}

		l.Infow(
			fmt.Sprintf("[Request Middleware] REQUEST: uri: %v, status: %v\n", req.RequestURI, status),
			logger.Fields{
				"uri":           req.RequestURI,
				"status":        status,
				"id":            id,
				"remote_ip":     c.ClientIP(),
				"host":          req.Host,
				"method":        req.Method,
				"user_agent":    req.UserAgent(),
				"error":         requestErr,
				"latency":       latency.Nanoseconds(),
				"latency_human": latency.String(),
				"bytes_in":      req.ContentLength,
				"bytes_out":     size,
			},
		)

		fields := logger.Fields{
			"remote_ip":  c.ClientIP(),
			"latency":    latency.String(),
			"host":       req.Host,
			"request":    fmt.Sprintf("%s %s", req.Method, req.RequestURI),
			"status":     status,
			"size":       size,
			"user_agent": req.UserAgent(),
			"request_id": id,
		}

		switch {
		case status >= 500:
			l.Errorw("GinServer logger middleware: Server error", fields)
		case status >= 400:
			l.Errorw("GinServer logger middleware: Client error", fields)
		case status >= 300:
			l.Errorw("GinServer logger middleware: Redirection", fields)
		default:
			l.Infow("GinServer logger middleware: Success", fields)
		}
	}
}
