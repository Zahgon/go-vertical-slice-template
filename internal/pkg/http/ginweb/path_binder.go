package ginweb

import (
	"encoding"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ParamTag is the struct tag naming the path parameter a field is bound from.
const ParamTag = "param"

// JoinCatchAll folds the remainder captured by a trailing catch all segment
// back into the named parameter, so the parameter matches greedily across
// slashes instead of stopping at the first one.
func JoinCatchAll(c *gin.Context, name string, catchAll string) {
	rest := c.Param(catchAll)
	if rest == "" {
		return
	}

	for i := range c.Params {
		if c.Params[i].Key == name {
			c.Params[i].Value += rest

			return
		}
	}
}

// BindPathParams populates the fields of i that carry a param tag from the
// matching path parameters. A conversion failure is reported as a bad request
// carrying the underlying error, both as the message and as the internal
// cause.
func BindPathParams(c *gin.Context, i interface{}) error {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return nil
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return nil
	}

	valueType := value.Type()
	for index := 0; index < valueType.NumField(); index++ {
		field := valueType.Field(index)
		name := strings.Split(field.Tag.Get(ParamTag), ",")[0]
		if name == "" || name == "-" || !value.Field(index).CanSet() {
			continue
		}

		raw := c.Param(name)
		if raw == "" {
			continue
		}

		if err := setParamValue(raw, value.Field(index)); err != nil {
			return webcore.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	}

	return nil
}

func setParamValue(raw string, field reflect.Value) error {
	if unmarshaler, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return unmarshaler.UnmarshalText([]byte(raw))
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
		parsed, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(parsed)
	}

	return nil
}
