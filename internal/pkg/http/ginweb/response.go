package ginweb

import (
	"encoding/json"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"

	"github.com/gin-gonic/gin"
)

// JSON writes i as a JSON document with the given status code.
//
// It deliberately streams through an *json.Encoder rather than marshalling to a
// buffer: the encoder terminates every document with a newline, and that newline
// is part of the response body this service has always produced.
func JSON(c *gin.Context, code int, i interface{}) error {
	c.Writer.Header().Set(webcore.HeaderContentType, MIMEApplicationJSON)
	c.Status(code)

	return json.NewEncoder(c.Writer).Encode(i)
}
