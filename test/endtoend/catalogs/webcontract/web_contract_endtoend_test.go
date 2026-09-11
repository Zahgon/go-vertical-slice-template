//go:build e2e

// Package webcontract pins the HTTP contract of the web layer.
//
// Every case here covers a seam where a web-framework mechanism is wired by
// hand rather than inherited from the endpoint code: routing and path
// parameters, automatic OPTIONS, the `Allow` header, error mapping, the
// problem-detail envelope, request correlation, compression, the body-size
// guard and the documentation routes. A plausible wrong choice at any of them
// still compiles and still leaves the inherited suites green, so the contract
// is asserted here against a real server over the wire.
package webcontract

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mehdihadeli/go-vertical-slice-template/test/testfixtures/integration"
)

type webContractE2ETest struct {
	*integration.IntegrationTestSharedFixture
}

func TestWebContractE2E(t *testing.T) {
	suite.Run(
		t,
		&webContractE2ETest{
			IntegrationTestSharedFixture: integration.NewIntegrationTestSharedFixture(t),
		},
	)
}

const (
	problemContentType = "application_exceptions/problem+json"
	jsonContentType    = "application/json"
)

type response struct {
	status  int
	header  http.Header
	rawBody []byte
}

// do issues a request without following redirects and without letting the
// transport negotiate compression on its own, so status, headers and raw bytes
// are exactly what the server put on the wire.
func (c *webContractE2ETest) do(
	method string,
	path string,
	body []byte,
	headers map[string]string,
) *response {
	c.T().Helper()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequest(method, c.BaseAddress+path, reader)
	require.NoError(c.T(), err)

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	httpResponse, err := client.Do(request)
	require.NoError(c.T(), err)

	defer func() { _ = httpResponse.Body.Close() }()

	raw, err := io.ReadAll(httpResponse.Body)
	require.NoError(c.T(), err)

	return &response{
		status:  httpResponse.StatusCode,
		header:  httpResponse.Header,
		rawBody: raw,
	}
}

func (c *webContractE2ETest) problemDetail(res *response) string {
	c.T().Helper()

	var problem struct {
		Status int    `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}

	require.NoError(c.T(), json.Unmarshal(res.rawBody, &problem))

	return problem.Detail
}

func validProduct() []byte {
	return []byte(`{"name":"Contract Product","description":"contract description","price":12.5}`)
}

func jsonHeaders() map[string]string {
	return map[string]string{"Content-Type": jsonContentType}
}

// The products slice must stay mounted at /api/v1/products, and a created
// product must come back as bare application/json — no charset parameter — with
// the trailing newline a streaming encoder leaves behind.
func (c *webContractE2ETest) Test_Products_Route_Is_Mounted_Under_Api_V1_And_Answers_Bare_Json() {
	res := c.do(http.MethodPost, "/api/v1/products", validProduct(), jsonHeaders())

	assert.Equal(c.T(), http.StatusCreated, res.status)
	assert.Equal(c.T(), jsonContentType, res.header.Get("Content-Type"))
	assert.True(c.T(),
		bytes.HasSuffix(res.rawBody, []byte("\n")),
		"created response must keep its trailing newline, got %q",
		string(res.rawBody),
	)
	assert.Contains(c.T(), string(res.rawBody), `"productId"`)
}

// An unknown route is mapped through the same error parser as everything else.
// The parser does not recognise the router's own not-found error, so it reports
// it as an internal server error while keeping the original wording in `detail`.
func (c *webContractE2ETest) Test_Unknown_Route_Is_Reported_As_Internal_Server_Error() {
	res := c.do(http.MethodGet, "/nope", nil, nil)

	assert.Equal(c.T(), http.StatusInternalServerError, res.status)
	assert.Equal(c.T(), problemContentType, res.header.Get("Content-Type"))
	assert.Equal(c.T(), "code=404, message=Not Found", c.problemDetail(res))
}

// A wrong method takes the same route, and the `Allow` header must list OPTIONS
// first — the order the source router produced, not alphabetical order.
func (c *webContractE2ETest) Test_Wrong_Method_Is_Reported_As_Internal_Server_Error_With_Allow_Header() {
	collection := c.do(http.MethodGet, "/api/v1/products", nil, nil)

	assert.Equal(c.T(), http.StatusInternalServerError, collection.status)
	assert.Equal(c.T(), problemContentType, collection.header.Get("Content-Type"))
	assert.Equal(c.T(), "OPTIONS, POST", collection.header.Get("Allow"))
	assert.Equal(c.T(), "code=405, message=Method Not Allowed", c.problemDetail(collection))

	byID := c.do(http.MethodPut, "/api/v1/products/"+c.Items[0].ProductID.String(), nil, nil)

	assert.Equal(c.T(), http.StatusInternalServerError, byID.status)
	assert.Equal(c.T(), "OPTIONS, GET", byID.header.Get("Allow"))
}

// Every registered path answers OPTIONS by itself, with no body and the same
// `Allow` header. An unregistered path must still fall through to the
// not-found path rather than answering OPTIONS.
func (c *webContractE2ETest) Test_Options_Is_Answered_With_No_Content_And_Allow_Header() {
	collection := c.do(http.MethodOptions, "/api/v1/products", nil, nil)

	assert.Equal(c.T(), http.StatusNoContent, collection.status)
	assert.Equal(c.T(), "OPTIONS, POST", collection.header.Get("Allow"))
	assert.Empty(c.T(), collection.rawBody)

	byID := c.do(http.MethodOptions, "/api/v1/products/"+c.Items[0].ProductID.String(), nil, nil)

	assert.Equal(c.T(), http.StatusNoContent, byID.status)
	assert.Equal(c.T(), "OPTIONS, GET", byID.header.Get("Allow"))
	assert.Empty(c.T(), byID.rawBody)

	unknown := c.do(http.MethodOptions, "/nope", nil, nil)

	assert.Equal(c.T(), http.StatusInternalServerError, unknown.status)
}

// The compression middleware advertises itself on every request it handles,
// whether or not it ends up compressing, and exactly once.
func (c *webContractE2ETest) Test_Vary_Header_Is_Advertised_Once_On_Handled_Requests() {
	res := c.do(http.MethodGet, "/nope", nil, nil)

	assert.Equal(c.T(), []string{"Accept-Encoding"}, res.header.Values("Vary"))
}

// The diagnostic paths are skipped by the whole chain — no logging, no
// compression, no error rendering. Nothing writes a response for them, so they
// answer an empty 200 even though no route is registered.
func (c *webContractE2ETest) Test_Skipped_Paths_Answer_Empty_Ok_Without_Vary() {
	for _, path := range []string{"/health", "/metrics", "/favicon.ico", "/swagger"} {
		res := c.do(http.MethodGet, path, nil, nil)

		assert.Equal(c.T(), http.StatusOK, res.status, "path %s", path)
		assert.Empty(c.T(), res.rawBody, "path %s", path)
		assert.Empty(c.T(), res.header.Values("Vary"), "path %s", path)
	}
}

// Compression must survive the error path too: problem details are rendered
// while the compressing writer is installed, not after it has been unwound.
func (c *webContractE2ETest) Test_Response_Is_Gzip_Framed_When_The_Client_Accepts_It() {
	res := c.do(
		http.MethodPost,
		"/api/v1/products",
		validProduct(),
		map[string]string{"Content-Type": jsonContentType, "Accept-Encoding": "gzip"},
	)

	assert.Equal(c.T(), http.StatusCreated, res.status)
	assert.Equal(c.T(), "gzip", res.header.Get("Content-Encoding"))
	assert.Equal(c.T(), []string{"Accept-Encoding"}, res.header.Values("Vary"))

	reader, err := gzip.NewReader(bytes.NewReader(res.rawBody))
	require.NoError(c.T(), err, "body must really be gzip framed")

	defer func() { _ = reader.Close() }()

	decoded, err := io.ReadAll(reader)
	require.NoError(c.T(), err)
	assert.Contains(c.T(), string(decoded), `"productId"`)

	failed := c.do(
		http.MethodGet,
		"/nope",
		nil,
		map[string]string{"Accept-Encoding": "gzip"},
	)

	assert.Equal(c.T(), "gzip", failed.header.Get("Content-Encoding"))
}

// A caller-supplied correlation id is echoed untouched; otherwise one is minted
// in the same 32-character alphanumeric shape the source produced.
func (c *webContractE2ETest) Test_Request_Id_Is_Echoed_When_Supplied_And_Minted_Otherwise() {
	supplied := c.do(
		http.MethodPost,
		"/api/v1/products",
		validProduct(),
		map[string]string{"Content-Type": jsonContentType, "X-Request-Id": "contract-fixed-id"},
	)

	assert.Equal(c.T(), "contract-fixed-id", supplied.header.Get("X-Request-Id"))

	minted := c.do(http.MethodPost, "/api/v1/products", validProduct(), jsonHeaders())

	assert.Regexp(c.T(), regexp.MustCompile(`^[A-Za-z0-9]{32}$`), minted.header.Get("X-Request-Id"))
}

// The body-size guard sits outside request correlation and compression, so an
// oversized request is refused before either runs. It must still be rendered as
// a problem detail, which only happens if an error raised that far out reaches
// the renderer.
func (c *webContractE2ETest) Test_Oversized_Body_Is_Refused_As_Request_Entity_Too_Large() {
	oversized := []byte(
		`{"name":"Big","description":"` + strings.Repeat("x", 3_000_000) + `","price":1.5}`,
	)

	res := c.do(http.MethodPost, "/api/v1/products", oversized, jsonHeaders())

	assert.Equal(c.T(), http.StatusInternalServerError, res.status)
	assert.Equal(c.T(), problemContentType, res.header.Get("Content-Type"))
	assert.Equal(c.T(), "code=413, message=Request Entity Too Large", c.problemDetail(res))
	assert.Empty(c.T(), res.header.Values("Vary"), "the guard runs before compression")
	assert.Empty(c.T(), res.header.Get("X-Request-Id"), "the guard runs before request correlation")
}

// Path parameters are read from the escaped path, so an encoded slash stays
// encoded while an encoded space is decoded. Getting this backwards silently
// changes which requests reach the handler.
func (c *webContractE2ETest) Test_Path_Parameter_Keeps_Encoded_Slash_And_Decodes_Space() {
	encodedSlash := c.do(http.MethodGet, "/api/v1/products/aaa%2Fbbb", nil, nil)

	assert.Equal(c.T(), http.StatusBadRequest, encodedSlash.status)
	assert.Contains(c.T(), c.problemDetail(encodedSlash), "aaa%2Fbbb")

	encodedSpace := c.do(http.MethodGet, "/api/v1/products/abc%20def", nil, nil)

	assert.Equal(c.T(), http.StatusBadRequest, encodedSpace.status)
	assert.Contains(c.T(), c.problemDetail(encodedSpace), "abc def")
}

// The parameter matches greedily across slashes, and neither a trailing slash
// nor a doubled slash is redirected away.
func (c *webContractE2ETest) Test_Path_Parameter_Matches_Greedily_And_No_Path_Is_Redirected() {
	greedy := c.do(http.MethodGet, "/api/v1/products/x/y", nil, nil)

	assert.Equal(c.T(), http.StatusBadRequest, greedy.status)
	assert.Contains(c.T(), c.problemDetail(greedy), "x/y")

	trailingSlash := c.do(http.MethodGet, "/api/v1/products/", nil, nil)

	assert.Equal(c.T(), http.StatusInternalServerError, trailingSlash.status)
	assert.Equal(c.T(), "code=404, message=Not Found", c.problemDetail(trailingSlash))

	doubledSlash := c.do(http.MethodGet, "//api/v1/products", nil, nil)

	assert.Equal(c.T(), http.StatusInternalServerError, doubledSlash.status)
	assert.Equal(c.T(), "code=404, message=Not Found", c.problemDetail(doubledSlash))
}

// The documentation routes keep their own quirks: the root redirect duplicates
// the separator, and a missing asset is answered by the file server rather than
// by the router.
func (c *webContractE2ETest) Test_Documentation_Routes_Keep_Their_Redirect_And_Not_Found_Bodies() {
	root := c.do(http.MethodGet, "/swagger/", nil, nil)

	assert.Equal(c.T(), http.StatusMovedPermanently, root.status)
	assert.Equal(c.T(), "/swagger//index.html", root.header.Get("Location"))
	assert.Empty(c.T(), root.rawBody)

	missing := c.do(http.MethodGet, "/swagger/does-not-exist.js", nil, nil)

	assert.Equal(c.T(), http.StatusNotFound, missing.status)
	assert.Equal(c.T(), "404 page not found\n", string(missing.rawBody))

	spec := c.do(http.MethodGet, "/swagger/doc.json", nil, nil)

	assert.Equal(c.T(), http.StatusOK, spec.status)
	assert.Contains(c.T(), string(spec.rawBody), `"/api/v1/products"`)
}

// Binding and validation failures are reported through the same envelope, and
// the unsupported-media-type case is reported as a bad request — the status the
// endpoint wraps it with — while the original code survives inside `detail`.
func (c *webContractE2ETest) Test_Binding_And_Validation_Failures_Share_The_Problem_Envelope() {
	validation := c.do(http.MethodPost, "/api/v1/products", []byte(`{}`), jsonHeaders())

	assert.Equal(c.T(), http.StatusBadRequest, validation.status)
	assert.Equal(c.T(), problemContentType, validation.header.Get("Content-Type"))
	assert.Contains(c.T(), c.problemDetail(validation), "validation error")

	malformed := c.do(http.MethodPost, "/api/v1/products", []byte(`{"name":`), jsonHeaders())

	assert.Equal(c.T(), http.StatusBadRequest, malformed.status)
	assert.Contains(c.T(), c.problemDetail(malformed), "error in the binding request")

	unsupported := c.do(
		http.MethodPost,
		"/api/v1/products",
		validProduct(),
		map[string]string{"Content-Type": "text/plain"},
	)

	assert.Equal(c.T(), http.StatusBadRequest, unsupported.status)
	assert.Contains(c.T(),
		c.problemDetail(unsupported),
		fmt.Sprintf("code=%d, message=Unsupported Media Type", http.StatusUnsupportedMediaType),
	)
}
