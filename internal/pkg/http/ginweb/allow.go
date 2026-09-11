package ginweb

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AllowHeader renders an `Allow` header value listing OPTIONS first and then the
// remaining methods in the order they were registered.
func AllowHeader(methods []string) string {
	allow := make([]string, 0, len(methods)+1)
	allow = append(allow, http.MethodOptions)

	for _, method := range methods {
		method = strings.TrimSpace(method)
		if method == "" || method == http.MethodOptions {
			continue
		}
		allow = append(allow, method)
	}

	return strings.Join(allow, ", ")
}

// RegisterAutoOptions answers OPTIONS on every registered path with 204 and an
// `Allow` header, so callers can discover a route's methods without the service
// declaring an OPTIONS handler for each one. Call it once, after every route has
// been mapped. Paths that were never registered stay unknown and fall through to
// the not-found path.
func RegisterAutoOptions(engine *gin.Engine) {
	paths := make([]string, 0)
	methods := make(map[string][]string)

	for _, route := range engine.Routes() {
		if _, seen := methods[route.Path]; !seen {
			paths = append(paths, route.Path)
		}
		methods[route.Path] = append(methods[route.Path], route.Method)
	}

	for _, path := range paths {
		registered := methods[path]
		if containsMethod(registered, http.MethodOptions) {
			continue
		}

		allow := AllowHeader(registered)
		engine.OPTIONS(path, func(c *gin.Context) {
			c.Header("Allow", allow)
			c.Status(http.StatusNoContent)
		})
	}
}

func containsMethod(methods []string, method string) bool {
	for _, candidate := range methods {
		if candidate == method {
			return true
		}
	}

	return false
}
