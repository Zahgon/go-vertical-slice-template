package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
)

const gzipScheme = "gzip"

// Gzip compresses the response body when the client advertises support for it.
// The `Vary` header is emitted for every handled request, whether or not the
// body ends up compressed, so caches key on the request encoding either way.
func Gzip(level int, skipper webcore.Skipper) gin.HandlerFunc {
	if skipper == nil {
		skipper = webcore.DefaultSkipper
	}

	return func(c *gin.Context) {
		if skipper(c) {
			c.Next()

			return
		}

		c.Writer.Header().Add(webcore.HeaderVary, webcore.HeaderAcceptEncoding)

		if !strings.Contains(c.Request.Header.Get(webcore.HeaderAcceptEncoding), gzipScheme) {
			c.Next()

			return
		}

		original := c.Writer

		writer, err := gzip.NewWriterLevel(original, level)
		if err != nil {
			c.Next()

			return
		}

		original.Header().Set(webcore.HeaderContentEncoding, gzipScheme)

		compressed := &gzipResponseWriter{ResponseWriter: original, writer: writer}
		c.Writer = compressed

		defer func() {
			if !compressed.wroteBody {
				if original.Header().Get(webcore.HeaderContentEncoding) == gzipScheme {
					original.Header().Del(webcore.HeaderContentEncoding)
				}

				c.Writer = original
				writer.Reset(io.Discard)
			}

			_ = writer.Close()
		}()

		c.Next()
	}
}

// gzipResponseWriter streams everything the handlers write through the gzip
// writer. `Content-Length` is dropped because the compressed length is not
// known until the stream is closed.
type gzipResponseWriter struct {
	gin.ResponseWriter

	writer    *gzip.Writer
	wroteBody bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.Header().Del(webcore.HeaderContentLength)
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get(webcore.HeaderContentType) == "" {
		w.Header().Set(webcore.HeaderContentType, http.DetectContentType(b))
	}

	w.wroteBody = true
	w.WriteHeaderNow()

	return w.writer.Write(b)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipResponseWriter) Flush() {
	_ = w.writer.Flush()
	w.ResponseWriter.Flush()
}
