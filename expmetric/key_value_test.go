package expmetric

import (
	"expvar"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

type testJSONValue struct {
	Path string `json:"name"`
}

func TestKeyValueStruct(t *testing.T) {
	app := iris.New()

	var (
		keyFunc = func(ctx iris.Context) string {
			return "test-key"
		}

		valueFunc = func(ctx iris.Context) (interface{}, bool) {
			value := testJSONValue{
				Path: ctx.Path(),
			}
			return value, true
		}

		expectedResponseBody = `{"test-key": {"name":"/test-prefix/path"}}`

		metricMiddleware = KeyValue(keyFunc, valueFunc, MetricName("test-metric"))
	)

	handler := func(ctx iris.Context) {
		ctx.ContentType("application/json")

		variable := expvar.Get("test-metric").String()
		ctx.WriteString(variable)
	}

	app.Get("/test-prefix/path", metricMiddleware, handler)

	e := httptest.New(t, app)
	e.GET("/test-prefix/path").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").
		Body().IsEqual(expectedResponseBody)
}
