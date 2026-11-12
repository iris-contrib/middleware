package tollboothic_test

import (
	"net/http"
	"testing"

	"github.com/iris-contrib/middleware/tollboothic"

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// newApp returns an application with one limited and one free route.
//
// tollbooth keys its buckets on the client address and does not limit a
// request that has none. The in-memory test client has none, so the address
// is taken from a request header here.
func newApp(l *limiter.Limiter) *iris.Application {
	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		if addr := ctx.Header("X-Test-Addr"); addr != "" {
			ctx.Request().RemoteAddr = addr + ":54321"
		}
		ctx.Next()
	})

	app.Get("/", tollboothic.LimitHandler(l), func(ctx iris.Context) {
		ctx.WriteString("ok")
	})
	app.Get("/free", func(ctx iris.Context) {
		ctx.WriteString("free")
	})

	return app
}

// One request per second with a burst of one: the second request
// of the same client inside that second is over the limit.
func TestLimitHandler(t *testing.T) {
	l := tollbooth.NewLimiter(1, nil)
	e := httptest.New(t, newApp(l))

	r := e.GET("/").WithHeader("X-Test-Addr", "192.0.2.1").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("ok")
	r.Header("X-Rate-Limit-Limit").IsEqual("1.00")

	r = e.GET("/").WithHeader("X-Test-Addr", "192.0.2.1").Expect().Status(httptest.StatusTooManyRequests)
	r.Body().IsEqual("You have reached maximum request limit.")
	r.HasContentType("text/plain", "utf-8")
	r.Header("X-Rate-Limit-Limit").IsEqual("1.00")
	r.Header("X-Rate-Limit-Request-Remote-Addr").IsEqual("192.0.2.1:54321")

	// Another client has its own bucket.
	e.GET("/").WithHeader("X-Test-Addr", "192.0.2.2").Expect().Status(httptest.StatusOK)

	// A route without the handler is not limited.
	for range 3 {
		e.GET("/free").WithHeader("X-Test-Addr", "192.0.2.1").Expect().Status(httptest.StatusOK)
	}
}

// The answer for a limited request comes from the limiter's configuration.
func TestLimitHandlerMessage(t *testing.T) {
	l := tollbooth.NewLimiter(1, nil)
	l.SetMessage(`{"error":"slow down"}`)
	l.SetMessageContentType("application/json; charset=utf-8")
	l.SetStatusCode(http.StatusServiceUnavailable)

	e := httptest.New(t, newApp(l))

	e.GET("/").WithHeader("X-Test-Addr", "192.0.2.1").Expect().Status(httptest.StatusOK)

	r := e.GET("/").WithHeader("X-Test-Addr", "192.0.2.1").Expect().Status(httptest.StatusServiceUnavailable)
	r.HasContentType("application/json", "utf-8")
	r.JSON().Object().HasValue("error", "slow down")
}

// The handler that follows a limited request does not run.
func TestLimitHandlerStopsTheChain(t *testing.T) {
	l := tollbooth.NewLimiter(1, nil)

	var served int

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.Request().RemoteAddr = "192.0.2.1:54321"
		ctx.Next()
	})
	app.Get("/", tollboothic.LimitHandler(l), func(ctx iris.Context) {
		served++
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK)
	e.GET("/").Expect().Status(httptest.StatusTooManyRequests)
	e.GET("/").Expect().Status(httptest.StatusTooManyRequests)

	if served != 1 {
		t.Fatalf("expected the route handler to run once, it ran %d times", served)
	}
}
