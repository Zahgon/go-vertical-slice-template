//go:build unit

package ginweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uuid "github.com/satori/go.uuid"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
)

type productBody struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type formBody struct {
	Name    string  `form:"name"`
	Enabled bool    `form:"enabled"`
	Count   int     `form:"count"`
	Ratio   float64 `form:"ratio"`
	Skipped string
	Ignored string `form:"-"`
}

type pathBody struct {
	ProductID uuid.UUID `param:"id"`
	Revision  int       `param:"rev"`
}

// requestContext builds a gin context carrying the given request, which is all
// BindBody reads.
func requestContext(method string, contentType string, body string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(method, "/api/v1/products", strings.NewReader(body))
	if contentType == "" {
		req.Header.Del("Content-Type")
	} else {
		req.Header.Set("Content-Type", contentType)
	}
	req.ContentLength = int64(len(body))
	c.Request = req

	return c
}

func httpErrorFrom(t *testing.T, err error) *webcore.HTTPError {
	t.Helper()
	require.Error(t, err)
	httpErr, ok := err.(*webcore.HTTPError)
	require.Truef(t, ok, "expected *webcore.HTTPError, got %T", err)

	return httpErr
}

func Test_BindBody_Binds_A_Well_Formed_Json_Document(t *testing.T) {
	request := &productBody{}

	err := BindBody(requestContext(http.MethodPost, "application/json", `{"name":"Book","description":"A book","price":12.5}`), request)

	require.NoError(t, err)
	assert.Equal(t, "Book", request.Name)
	assert.Equal(t, "A book", request.Description)
	assert.Equal(t, 12.5, request.Price)
}

func Test_BindBody_Accepts_A_Json_Content_Type_Carrying_A_Charset(t *testing.T) {
	request := &productBody{}

	err := BindBody(requestContext(http.MethodPost, "application/json; charset=utf-8", `{"name":"Book"}`), request)

	require.NoError(t, err)
	assert.Equal(t, "Book", request.Name)
}

func Test_BindBody_Reports_A_Json_Syntax_Error_As_A_Bad_Request(t *testing.T) {
	err := BindBody(requestContext(http.MethodPost, "application/json", `{"name":"Book",}`), &productBody{})

	httpErr := httpErrorFrom(t, err)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	assert.Contains(t, httpErr.Error(), "code=400, message=Syntax error: offset=")
	assert.NotNil(t, httpErr.Unwrap())
}

func Test_BindBody_Reports_A_Json_Type_Error_With_The_Offending_Field(t *testing.T) {
	err := BindBody(requestContext(http.MethodPost, "application/json", `{"name":"Book","price":"free"}`), &productBody{})

	httpErr := httpErrorFrom(t, err)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	assert.Contains(t, httpErr.Error(), "Unmarshal type error: expected=float64, got=string, field=price, offset=")
}

func Test_BindBody_Reports_A_Truncated_Json_Document_As_An_Unexpected_End(t *testing.T) {
	err := BindBody(requestContext(http.MethodPost, "application/json", `{"name":`), &productBody{})

	assert.Equal(t, "code=400, message=unexpected EOF, internal=unexpected EOF", httpErrorFrom(t, err).Error())
}

func Test_BindBody_Leaves_The_Request_Untouched_When_There_Is_No_Body(t *testing.T) {
	request := &productBody{}

	err := BindBody(requestContext(http.MethodPost, "application/json", ""), request)

	require.NoError(t, err)
	assert.Equal(t, &productBody{}, request)
}

func Test_BindBody_Refuses_A_Body_Whose_Media_Type_Is_Absent_Or_Unknown(t *testing.T) {
	for _, contentType := range []string{"", "text/plain", "not/a-real-type"} {
		err := BindBody(requestContext(http.MethodPost, contentType, `{"name":"Book"}`), &productBody{})

		httpErr := httpErrorFrom(t, err)
		assert.Equalf(t, http.StatusUnsupportedMediaType, httpErr.Code, "content type %q", contentType)
		assert.Equalf(t, "code=415, message=Unsupported Media Type", httpErr.Error(), "content type %q", contentType)
	}
}

func Test_BindBody_Matches_Xml_Elements_Against_Field_Names_Case_Sensitively(t *testing.T) {
	lowercase := &productBody{}
	require.NoError(t, BindBody(requestContext(http.MethodPost, "application/xml", `<product><name>Book</name></product>`), lowercase))
	assert.Empty(t, lowercase.Name, "a lowercase element must not reach the exported field")

	exact := &productBody{}
	require.NoError(t, BindBody(requestContext(http.MethodPost, "text/xml", `<product><Name>Book</Name></product>`), exact))
	assert.Equal(t, "Book", exact.Name)
}

func Test_BindBody_Reports_Malformed_Xml_As_A_Bad_Request(t *testing.T) {
	err := BindBody(requestContext(http.MethodPost, "application/xml", `<product><Name>Book`), &productBody{})

	httpErr := httpErrorFrom(t, err)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	assert.Contains(t, httpErr.Error(), "code=400, message=")
}

func Test_BindBody_Binds_Form_Fields_Only_Through_An_Explicit_Form_Tag(t *testing.T) {
	request := &formBody{}
	body := "name=Book&enabled=true&count=7&ratio=1.5&Skipped=nope&Ignored=nope"

	err := BindBody(requestContext(http.MethodPost, "application/x-www-form-urlencoded", body), request)

	require.NoError(t, err)
	assert.Equal(t, "Book", request.Name)
	assert.True(t, request.Enabled)
	assert.Equal(t, 7, request.Count)
	assert.Equal(t, 1.5, request.Ratio)
	assert.Empty(t, request.Skipped, "an untagged field is never matched by its Go name")
	assert.Empty(t, request.Ignored, "a field tagged - is skipped")
}

func Test_BindBody_Binds_Nothing_For_A_Form_Payload_Whose_Target_Has_No_Form_Tags(t *testing.T) {
	request := &productBody{}

	err := BindBody(requestContext(http.MethodPost, "application/x-www-form-urlencoded", "name=Book&description=A+book&price=3"), request)

	require.NoError(t, err)
	assert.Equal(t, &productBody{}, request)
}

func Test_BindPathParams_Binds_Through_The_Text_Unmarshaler_Of_The_Field(t *testing.T) {
	id := uuid.NewV4()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: id.String()}, {Key: "rev", Value: "42"}}
	request := &pathBody{}

	err := BindPathParams(c, request)

	require.NoError(t, err)
	assert.Equal(t, id, request.ProductID)
	assert.Equal(t, 42, request.Revision)
}

func Test_BindPathParams_Reports_An_Unparseable_Parameter_As_A_Bad_Request(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

	err := BindPathParams(c, &pathBody{})

	assert.Equal(
		t,
		"code=400, message=uuid: incorrect UUID length: not-a-uuid, internal=uuid: incorrect UUID length: not-a-uuid",
		httpErrorFrom(t, err).Error(),
	)
}

func Test_JoinCatchAll_Folds_The_Remainder_Back_Into_The_Named_Parameter(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: "x"}, {Key: "action", Value: "/y/z"}}

	JoinCatchAll(c, "id", "action")

	assert.Equal(t, "x/y/z", c.Param("id"))
}

func Test_JoinCatchAll_Leaves_The_Parameter_Alone_When_Nothing_Was_Caught(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: "x"}}

	JoinCatchAll(c, "id", "action")

	assert.Equal(t, "x", c.Param("id"))
}
