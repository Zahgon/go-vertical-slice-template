package bodylimit

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
)

// BodyLimit refuses request bodies larger than limit, expressed the way the
// configuration states it ("2M", "512K", "1024"). The declared content length is
// rejected up front; a request that lies about its size is cut off while being
// read, so an oversized stream cannot be used to exhaust memory.
func BodyLimit(limit string) gin.HandlerFunc {
	max, err := parseSize(limit)
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		if c.Request.ContentLength > max {
			_ = c.Error(webcore.ErrStatusRequestEntityTooLarge)
			c.Abort()

			return
		}

		if c.Request.Body != nil {
			c.Request.Body = &limitedReader{reader: c.Request.Body, limit: max}
		}

		c.Next()
	}
}

type limitedReader struct {
	reader io.ReadCloser
	limit  int64
	read   int64
}

func (r *limitedReader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)
	r.read += int64(n)

	if r.read > r.limit {
		return n, webcore.ErrStatusRequestEntityTooLarge
	}

	return n, err
}

func (r *limitedReader) Close() error {
	return r.reader.Close()
}

// parseSize turns a human readable size into a byte count.
func parseSize(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, webcore.NewHTTPError(http.StatusInternalServerError, "empty body limit")
	}

	multiplier := int64(1)
	switch strings.ToUpper(value[len(value)-1:]) {
	case "G":
		multiplier = 1 << 30
	case "M":
		multiplier = 1 << 20
	case "K":
		multiplier = 1 << 10
	case "B":
		multiplier = 1
	default:
		multiplier = 0
	}

	digits := value
	if multiplier != 0 {
		digits = strings.TrimSpace(value[:len(value)-1])
	} else {
		multiplier = 1
	}

	size, err := strconv.ParseFloat(digits, 64)
	if err != nil {
		return 0, err
	}

	return int64(size * float64(multiplier)), nil
}
