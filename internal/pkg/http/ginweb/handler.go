package ginweb

import (
	"github.com/gin-gonic/gin"
)

// HandlerFunc is a request handler that reports a failure by returning an
// error instead of writing it, leaving the problem detail middleware to render
// the response.
type HandlerFunc func(c *gin.Context) error

// Handler adapts a HandlerFunc to gin by recording a returned error on the
// context and stopping the handler chain, so the error is rendered exactly
// once while the surrounding middlewares still unwind.
func Handler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c); err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}
}
