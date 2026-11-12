package prometheus_test

import (
	"strings"
	"testing"

	prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// scrape returns what a Prometheus server would read from the application.
func scrape(e *httptest.Expect) string {
	return e.GET("/metrics").Expect().Status(httptest.StatusOK).Body().Raw()
}

func expectLines(t *testing.T, metrics string, lines ...string) {
	t.Helper()

	for _, line := range lines {
		if !strings.Contains(metrics, line+"\n") {
			t.Errorf("expected the metrics to contain the line:\n%s", line)
		}
	}

	if t.Failed() {
		t.Logf("the metrics of the middleware were:\n%s", middlewareLines(metrics))
	}
}

func middlewareLines(metrics string) string {
	var b strings.Builder
	for line := range strings.SplitSeq(metrics, "\n") {
		if strings.HasPrefix(line, "http_request") {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func TestServeHTTP(t *testing.T) {
	m := prometheusMiddleware.New("test_service", 0.5, 2)

	app := iris.New()
	// On the root application Use also runs for requests that match no route.
	app.Use(m.ServeHTTP)

	app.Get("/", func(ctx iris.Context) { ctx.WriteString("index") })
	app.Post("/items", func(ctx iris.Context) { ctx.StatusCode(iris.StatusCreated) })
	app.Get("/fail", func(ctx iris.Context) { ctx.StopWithStatus(iris.StatusInternalServerError) })
	app.Get("/metrics", iris.FromStd(promhttp.Handler()))

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK)
	e.GET("/").Expect().Status(httptest.StatusOK)
	e.POST("/items").Expect().Status(httptest.StatusCreated)
	e.GET("/fail").Expect().Status(httptest.StatusInternalServerError)
	e.GET("/missing").Expect().Status(httptest.StatusNotFound)

	metrics := scrape(e)
	expectLines(t, metrics,
		`http_requests_total{code="200",method="GET",path="/",service="test_service"} 2`,
		`http_requests_total{code="201",method="POST",path="/items",service="test_service"} 1`,
		`http_requests_total{code="500",method="GET",path="/fail",service="test_service"} 1`,
		// A not found is counted, and counted once.
		`http_requests_total{code="404",method="GET",path="/missing",service="test_service"} 1`,

		`http_request_duration_seconds_count{code="200",method="GET",path="/",service="test_service"} 2`,
		// The two requests took well under half a second, the first of the given buckets.
		`http_request_duration_seconds_bucket{code="200",method="GET",path="/",service="test_service",le="0.5"} 2`,
		`http_request_duration_seconds_bucket{code="200",method="GET",path="/",service="test_service",le="2"} 2`,
		`http_request_duration_seconds_bucket{code="200",method="GET",path="/",service="test_service",le="+Inf"} 2`,
	)

	// The scrape itself is recorded once it has been served.
	expectLines(t, scrape(e),
		`http_requests_total{code="200",method="GET",path="/metrics",service="test_service"} 1`,
	)
}

// Registered on a single route the middleware records that route only.
func TestServeHTTPPerRoute(t *testing.T) {
	m := prometheusMiddleware.New("per_route_service")

	app := iris.New()
	app.Get("/measured", m.ServeHTTP, func(ctx iris.Context) { ctx.WriteString("ok") })
	app.Get("/other", func(ctx iris.Context) { ctx.WriteString("ok") })
	app.Get("/metrics", iris.FromStd(promhttp.Handler()))

	e := httptest.New(t, app)
	e.GET("/measured").Expect().Status(httptest.StatusOK)
	e.GET("/other").Expect().Status(httptest.StatusOK)
	e.GET("/missing").Expect().Status(httptest.StatusNotFound)

	metrics := scrape(e)
	expectLines(t, metrics,
		`http_requests_total{code="200",method="GET",path="/measured",service="per_route_service"} 1`,
		// No buckets were given, so the default ones are used.
		`http_request_duration_seconds_bucket{code="200",method="GET",path="/measured",service="per_route_service",le="0.3"} 1`,
		`http_request_duration_seconds_bucket{code="200",method="GET",path="/measured",service="per_route_service",le="1.2"} 1`,
		`http_request_duration_seconds_bucket{code="200",method="GET",path="/measured",service="per_route_service",le="5"} 1`,
	)

	for _, path := range []string{"/other", "/missing"} {
		if strings.Contains(metrics, `path="`+path+`",service="per_route_service"`) {
			t.Errorf("expected no metrics for %s", path)
		}
	}
}

// The collectors are registered to the default registry,
// which accepts a service name once.
func TestNewTwiceWithTheSameName(t *testing.T) {
	prometheusMiddleware.New("twice_service")

	defer func() {
		if recover() == nil {
			t.Fatal("expected a second New with the same name to panic")
		}
	}()

	prometheusMiddleware.New("twice_service")
}
