package cors_test

import (
	"testing"
	"time"

	"github.com/iris-contrib/middleware/cors"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

func TestCorsAllowOrigins(t *testing.T) {
	origin := "https://iris-go.com"
	opts := cors.Options{
		AllowedOrigins: []string{origin},
		AllowedHeaders: []string{"Content-Type"},
		AllowedMethods: []string{"GET", "POST", "PUT", "HEAD"},
		ExposedHeaders: []string{"X-Header"},
		MaxAge:         int((24 * time.Hour).Seconds()),
		// Debug:          true,
	}

	app := iris.New()
	app.UseRouter(cors.New(opts))
	// OR per group of routes:
	// v1 := app.Party("/v1")
	// v1.AllowMethods(iris.MethodOptions)
	// v1.Use(cors.New(opts))

	h := func(ctx iris.Context) {
		ctx.Writef("%s: %s", ctx.Method(), ctx.Path())
	}

	app.Get("/", h)
	app.Post("/", h)
	app.Patch("/", h)

	e := httptest.New(t, app) //, httptest.LogLevel("debug")) //, httptest.Debug(true))

	// test origin empty.
	r := e.GET("/").Expect().Status(httptest.StatusOK)
	r.Body().Equal("GET: /")
	r.Headers().NotContainsKey("Access-Control-Allow-Origin").
		NotContainsKey("Access-Control-Allow-Credentials").NotContainsKey("Access-Control-Expose-Headers")

	// test allow.
	r = e.GET("/").WithHeader("Origin", origin).Expect().Status(httptest.StatusOK)
	r.Body().Equal("GET: /")
	r.Header("Access-Control-Allow-Origin").Equal(origin)
	r.Headers().NotContainsKey("Access-Control-Allow-Credentials")
	r.Header("Access-Control-Expose-Headers").Equal("X-Header")

	// test disallow, note the "http" instead of "https".
	r = e.GET("/").WithHeader("Origin", "http://iris-go.com").Expect().Status(httptest.StatusForbidden)
	r.Headers().NotContainsKey("Access-Control-Allow-Origin").
		NotContainsKey("Access-Control-Allow-Credentials").NotContainsKey("Access-Control-Expose-Headers")

	// test allow prefligh.
	r = e.OPTIONS("/").WithHeader("Origin", origin).
		WithHeader("Access-Control-Request-Method", "GET").
		WithHeader("Access-Control-Request-Headers", "Content-Type").
		Expect().Status(httptest.StatusOK)
	r.Header("Vary").Equal("Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
	r.Header("Access-Control-Allow-Origin").Equal(origin)
	r.Header("Access-Control-Allow-Credentials").Empty()
	// Spec says: Since the list of methods can be unbounded, simply returning the method indicated
	// by Access-Control-Request-Method (if supported) can be enough
	r.Header("Access-Control-Allow-Methods").Equal("GET")
	// Spec says: Since the list of headers can be unbounded, simply returning supported headers
	// from Access-Control-Request-Headers can be enough
	r.Header("Access-Control-Allow-Headers").Equal("Content-Type")
	r.Header("Access-Control-Max-Age").Equal("86400")

	// test no prefligh.
	r = e.OPTIONS("/").WithHeader("Origin", "http://github.com").
		WithHeader("Access-Control-Request-Method", "GET").Expect().Status(httptest.StatusForbidden)
	r.Header("Access-Control-Allow-Origin").Empty()
	r.Header("Access-Control-Allow-Credentials").Empty()
	r.Header("Access-Control-Allow-Methods").Empty()
	r.Header("Access-Control-Allow-Headers").Empty()
	r.Header("Access-Control-Max-Age").Empty()
}
