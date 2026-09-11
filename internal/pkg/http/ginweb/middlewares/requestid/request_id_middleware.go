package requestid

import (
	"math/rand"

	"github.com/gin-gonic/gin"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
)

const (
	idLength   = 32
	idAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// RequestID echoes the caller's request id back on the response, or mints one
// when the caller did not supply it, so a single request can be correlated
// across the access log and the client's own records.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.Request.Header.Get(webcore.HeaderXRequestID)
		if rid == "" {
			rid = generate(idLength)
		}

		c.Writer.Header().Set(webcore.HeaderXRequestID, rid)

		c.Next()
	}
}

func generate(length int) string {
	id := make([]byte, length)
	for i := range id {
		id[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}

	return string(id)
}
