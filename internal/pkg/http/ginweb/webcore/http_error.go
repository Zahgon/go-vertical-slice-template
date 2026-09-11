package webcore

import (
	"fmt"
	"net/http"
)

// HTTPError carries a status code and a message for failures raised by the web
// layer itself - routing misses, method mismatches, binding and body-size
// failures. Its `Error` text is what the problem-detail parser turns into the
// `detail` field of the response body, so the wording is part of the service's
// public contract.
type HTTPError struct {
	Code     int         `json:"-"`
	Message  interface{} `json:"message"`
	Internal error       `json:"-"`
}

// NewHTTPError builds an HTTPError, defaulting the message to the status text.
func NewHTTPError(code int, message ...interface{}) *HTTPError {
	err := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		err.Message = message[0]
	}

	return err
}

// SetInternal attaches the underlying error that caused this failure.
func (e *HTTPError) SetInternal(err error) *HTTPError {
	e.Internal = err

	return e
}

func (e *HTTPError) Error() string {
	if e.Internal == nil {
		return fmt.Sprintf("code=%d, message=%v", e.Code, e.Message)
	}

	return fmt.Sprintf("code=%d, message=%v, internal=%v", e.Code, e.Message, e.Internal)
}

func (e *HTTPError) Unwrap() error {
	return e.Internal
}

var (
	ErrNotFound                    = NewHTTPError(http.StatusNotFound)
	ErrMethodNotAllowed            = NewHTTPError(http.StatusMethodNotAllowed)
	ErrUnsupportedMediaType        = NewHTTPError(http.StatusUnsupportedMediaType)
	ErrStatusRequestEntityTooLarge = NewHTTPError(http.StatusRequestEntityTooLarge)
)
