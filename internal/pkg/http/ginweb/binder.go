package ginweb

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Supported request media types.
const (
	MIMEApplicationJSON = "application/json"
	MIMEApplicationXML  = "application/xml"
	MIMETextXML         = "text/xml"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
	MIMEMultipartForm   = "multipart/form-data"
)

// BindBody decodes the request body into i, choosing the decoder from the
// request's content type. A request without a body binds nothing and succeeds,
// which is what lets an empty payload fall through to validation instead of
// failing to parse. An unrecognised content type is refused with a 415, which
// the endpoints then re-wrap as a bad request.
func BindBody(c *gin.Context, i interface{}) error {
	req := c.Request
	if req == nil || req.Body == nil || req.ContentLength == 0 {
		return nil
	}

	ctype := req.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		return bindJSON(req.Body, i)
	case strings.HasPrefix(ctype, MIMEApplicationXML), strings.HasPrefix(ctype, MIMETextXML):
		return bindXML(req.Body, i)
	case strings.HasPrefix(ctype, MIMEApplicationForm), strings.HasPrefix(ctype, MIMEMultipartForm):
		return bindForm(c, i)
	default:
		return webcore.ErrUnsupportedMediaType
	}
}

func bindJSON(body io.Reader, i interface{}) error {
	err := json.NewDecoder(body).Decode(i)
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *json.UnmarshalTypeError:
		return webcore.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v",
				e.Type,
				e.Value,
				e.Field,
				e.Offset,
			),
		).SetInternal(err)
	case *json.SyntaxError:
		return webcore.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("Syntax error: offset=%v, error=%v", e.Offset, e.Error()),
		).SetInternal(err)
	}

	return webcore.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
}

func bindXML(body io.Reader, i interface{}) error {
	err := xml.NewDecoder(body).Decode(i)
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *xml.UnsupportedTypeError:
		return webcore.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("Unsupported type error: type=%v, error=%v", e.Type, e.Error()),
		).SetInternal(err)
	case *xml.SyntaxError:
		return webcore.NewHTTPError(
			http.StatusBadRequest,
			fmt.Sprintf("Syntax error: line=%v, error=%v", e.Line, e.Error()),
		).SetInternal(err)
	}

	return webcore.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
}

func bindForm(c *gin.Context, i interface{}) error {
	if err := c.Request.ParseForm(); err != nil {
		return webcore.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}

	if err := bindFormValues(i, c.Request.Form); err != nil {
		return webcore.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}

	return nil
}

// bindFormValues fills the fields that declare an explicit `form` tag. Fields
// without one are left untouched rather than matched on their Go name, so a dto
// that carries no form tags binds nothing at all.
func bindFormValues(i interface{}, values url.Values) error {
	val := reflect.ValueOf(i)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return nil
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for idx := 0; idx < typ.NumField(); idx++ {
		field := typ.Field(idx)

		name := field.Tag.Get("form")
		if name == "" || name == "-" {
			continue
		}
		name = strings.Split(name, ",")[0]

		raw, ok := values[name]
		if !ok || len(raw) == 0 {
			continue
		}

		if err := setFormValue(val.Field(idx), raw[0]); err != nil {
			return err
		}
	}

	return nil
}

func setFormValue(field reflect.Value, raw string) error {
	if !field.CanSet() {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		field.SetFloat(parsed)
	}

	return nil
}
