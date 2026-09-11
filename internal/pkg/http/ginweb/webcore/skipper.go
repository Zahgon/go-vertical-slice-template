package webcore

import (
	"github.com/gin-gonic/gin"
)

// Header names used across the web infrastructure.
const (
	HeaderXRequestID      = "X-Request-Id"
	HeaderContentType     = "Content-Type"
	HeaderContentLength   = "Content-Length"
	HeaderVary            = "Vary"
	HeaderAllow           = "Allow"
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderContentEncoding = "Content-Encoding"
	HeaderLocation        = "Location"
)

// Skipper defines a function to skip a middleware for the current request.
type Skipper func(c *gin.Context) bool

// DefaultSkipper never skips, so the middleware always runs.
func DefaultSkipper(*gin.Context) bool {
	return false
}
