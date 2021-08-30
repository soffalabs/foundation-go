package soffa

import (
	"fmt"
	"github.com/gavv/httpexpect/v2"
	"github.com/soffa-io/soffa-core-go/broker"
	"github.com/soffa-io/soffa-core-go/db"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type Tester struct {
	app    *App
	server *httptest.Server
	test   *testing.T
	expect *httpexpect.Expect
}

func NewTester(t *testing.T, app *App) Tester {
	app.bootstrap()
	server := httptest.NewServer(app.router.HttpHandler())
	return Tester{
		app:    app,
		test:   t,
		expect: httpexpect.New(t, server.URL),
		server: server,
	}
}

func (t *Tester) Truncate(models ...interface{}) {
	link := t.app.dbManager.GetLink()
	for _, model := range models {
		link.Truncate(model)
	}
}
func (t *Tester) TruncateN(linkId string, models ...interface{}) {
	link := t.app.dbManager.GetLinkN(linkId)
	for _, model := range models {
		link.Truncate(model)
	}
}

func (t *Tester) DB() *db.Manager {
	return t.app.dbManager
}

func (t *Tester) Publish(subj string, data interface{}) error {
	return t.app.broker.Publish(subj, data)
}

func (t *Tester) Subscribe(subj string, handler broker.Handler) {
	t.app.broker.Subscribe(subj, handler)
}

func (t *Tester) SubscribeAll(subj []string, handler broker.Handler) {
	for _, s := range subj {
		t.app.broker.Subscribe(s, handler)
	}
}

func (t *Tester) Close() {
	t.server.Close()
}


func (t *Tester) Arg(key string) interface{} {
	return t.app.args[key]
}
/*
func (t *Tester) Truncate(dsName string, names []string) {
	ds := t.App.GetDbLink(dsName)
	for _, name := range names {
		_ = ds.Exec("DELETE FROM " + ds.TableName(name))
	}
}*/

func (t *Tester) GET(path string) TestRequest {
	return TestRequest{
		request: t.expect.GET(path),
	}
}

func (t *Tester) POST(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.POST(path).WithJSON(data),
	}
}

func (t *Tester) PUT(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.PUT(path).WithJSON(data),
	}
}

func (t *Tester) DELETE(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.DELETE(path).WithJSON(data),
	}
}

func (t *Tester) PATCH(path string, data interface{}) TestRequest {
	return TestRequest{
		request: t.expect.PATCH(path).WithJSON(data),
	}
}


func (t TestRequest) Expect() TestResponse {
	return TestResponse{
		response: t.request.Expect(),
		test:     t.test,
	}
}

func (t TestRequest) WithNewBearer(subject string, audience string) TestRequest {
	secret := os.Getenv("JWT_SECRET")
	h.AssertNotEmpty(secret, "JWT_SECRET is missing")
	token, err := h.CreateJwt(secret, "app", subject, audience, h.Map{})
	assert.Nil(t.test, err)
	return t.Bearer(token)
}

func (t TestRequest) Bearer(token string) TestRequest {
	t.request.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	return t
}

func (t TestRequest) BasicAuth(user string, password string) TestRequest {
	t.request.WithBasicAuth(user, password)
	return t
}

type TestRequest struct {
	request *httpexpect.Request
	test    *testing.T
}

type TestResponse struct {
	response *httpexpect.Response
	test     *testing.T
}

type TestResult struct {
	value *httpexpect.Value
}

func (t TestResponse) OK() TestResponse {
	t.response.Status(http.StatusOK)
	return t
}

func (t TestResponse) Created() TestResponse {
	t.response.Status(http.StatusCreated)
	return t
}

func (t TestResponse) Unauthorized() TestResponse {
	t.response.Status(http.StatusUnauthorized)
	return t
}

func (t TestResponse) BadRequest() TestResponse {
	t.response.Status(http.StatusBadRequest)
	return t
}

func (t TestResponse) Forbidden() TestResponse {
	t.response.Status(http.StatusForbidden)
	return t
}

func (t TestResponse) Status(status int) TestResponse {
	t.response.Status(status)
	return t
}

func (t TestResponse) Json(path string) TestResult {
	return TestResult{
		value: t.response.JSON().Path(path),
	}
}

func (t TestResult) Is(value interface{}) TestResult {
	t.value.Equal(value)
	return t
}

func (t TestResult) Contains(value string) TestResult {
	t.value.String().Contains(value)
	return t
}

func (t TestResult) NotContains(value string) TestResult {
	t.value.String().NotContains(value)
	return t
}

func (t TestResult) IsArray() TestResult {
	t.value.Array()
	return t
}

func (t TestResult) IsEmptyArray() TestResult {
	t.value.Array().Empty()
	return t
}

func (t TestResult) IsNonEmptyArray() TestResult {
	t.value.Array().NotEmpty()
	return t
}

func (t TestResult) String() string {
	return t.value.String().Raw()
}

func (t TestResult) NotEmpty() TestResult {
	t.value.String().NotEmpty()
	return t
}

func (t TestResult) IsTrue() TestResult {
	t.value.Boolean().True()
	return t
}

func (t TestResult) Equal(value interface{}) TestResult {
	t.value.Equal(value)
	return t
}
