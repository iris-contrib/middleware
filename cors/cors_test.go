package cors_test

import (
	"testing"
	"time"

	"github.com/iris-contrib/middleware/cors"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
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
	// v1 := app.Router("/v1")
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
	r.Body().IsEqual("GET: /")
	r.Headers().NotContainsKey("Access-Control-Allow-Origin").
		NotContainsKey("Access-Control-Allow-Credentials").NotContainsKey("Access-Control-Expose-Headers")

	// test allow.
	r = e.GET("/").WithHeader("Origin", origin).Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("GET: /")
	r.Header("Access-Control-Allow-Origin").IsEqual(origin)
	r.Headers().NotContainsKey("Access-Control-Allow-Credentials")
	r.Header("Access-Control-Expose-Headers").IsEqual("X-Header")

	// test disallow, note the "http" instead of "https".
	r = e.GET("/").WithHeader("Origin", "http://iris-go.com").Expect().Status(httptest.StatusForbidden)
	r.Headers().NotContainsKey("Access-Control-Allow-Origin").
		NotContainsKey("Access-Control-Allow-Credentials").NotContainsKey("Access-Control-Expose-Headers")

	// test allow prefligh.
	r = e.OPTIONS("/").WithHeader("Origin", origin).
		WithHeader("Access-Control-Request-Method", "GET").
		WithHeader("Access-Control-Request-Headers", "Content-Type").
		Expect().Status(httptest.StatusOK)
	r.Header("Vary").IsEqual("Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
	r.Header("Access-Control-Allow-Origin").IsEqual(origin)
	r.Header("Access-Control-Allow-Credentials").IsEmpty()
	// Spec says: Since the list of methods can be unbounded, simply returning the method indicated
	// by Access-Control-Request-Method (if supported) can be enough
	r.Header("Access-Control-Allow-Methods").IsEqual("GET")
	// Spec says: Since the list of headers can be unbounded, simply returning supported headers
	// from Access-Control-Request-Headers can be enough
	r.Header("Access-Control-Allow-Headers").IsEqual("Content-Type")
	r.Header("Access-Control-Max-Age").IsEqual("86400")

	// test no prefligh.
	r = e.OPTIONS("/").WithHeader("Origin", "http://github.com").
		WithHeader("Access-Control-Request-Method", "GET").Expect().Status(httptest.StatusForbidden)
	r.Header("Access-Control-Allow-Origin").IsEmpty()
	r.Header("Access-Control-Allow-Credentials").IsEmpty()
	r.Header("Access-Control-Allow-Methods").IsEmpty()
	r.Header("Access-Control-Allow-Headers").IsEmpty()
	r.Header("Access-Control-Max-Age").IsEmpty()
}

// TestCorsVaryAppends proves the middleware adds to the Vary header instead of
// replacing it: a Vary an earlier middleware (compression, for one) already set
// must survive. In v14 `ctx.SetHeader` replaces, so the middleware uses
// `ctx.AddHeader` for this list-valued header.
func TestCorsVaryAppends(t *testing.T) {
	origin := "https://iris-go.com"
	opts := cors.Options{
		AllowedOrigins: []string{origin},
		AllowedMethods: []string{"GET"},
	}

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) { // stands in for the compression middleware.
		ctx.AddHeader("Vary", "Accept-Encoding")
		ctx.Next()
	})
	app.UseRouter(cors.New(opts))

	app.Get("/", func(ctx iris.Context) { ctx.WriteString("OK") })

	e := httptest.New(t, app)

	// the actual request: cors adds "Origin".
	r := e.GET("/").WithHeader("Origin", origin).Expect().Status(httptest.StatusOK)
	r.Headers().Value("Vary").Array().ConsistsOf("Accept-Encoding", "Origin")

	// the preflight request: cors adds the three request-header names.
	r = e.OPTIONS("/").WithHeader("Origin", origin).
		WithHeader("Access-Control-Request-Method", "GET").
		Expect().Status(httptest.StatusOK)
	r.Headers().Value("Vary").Array().ConsistsOf("Accept-Encoding",
		"Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
}

// Registered on a group of routes the middleware needs the group to answer
// OPTIONS as well, which Router.AllowMethods takes care of.
func TestCorsPerRouter(t *testing.T) {
	origin := "https://iris-go.com"

	app := iris.New()

	api := app.Router("/api")
	api.AllowMethods(iris.MethodOptions)
	api.Use(cors.New(cors.Options{
		AllowedOrigins: []string{origin},
		AllowedMethods: []string{"GET"},
	}))
	api.Get("/items", func(ctx iris.Context) { ctx.WriteString("items") })

	app.Get("/other", func(ctx iris.Context) { ctx.WriteString("other") })

	e := httptest.New(t, app)

	// the preflight request of a route of the group.
	r := e.OPTIONS("/api/items").WithHeader("Origin", origin).
		WithHeader("Access-Control-Request-Method", "GET").
		Expect().Status(httptest.StatusOK)
	r.Header("Access-Control-Allow-Origin").IsEqual(origin)
	r.Header("Access-Control-Allow-Methods").IsEqual("GET")
	r.Body().IsEmpty()

	// the actual request.
	r = e.GET("/api/items").WithHeader("Origin", origin).Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("items")
	r.Header("Access-Control-Allow-Origin").IsEqual(origin)

	// a route outside the group is not touched.
	r = e.GET("/other").WithHeader("Origin", origin).Expect().Status(httptest.StatusOK)
	r.Headers().NotContainsKey("Access-Control-Allow-Origin")
}

// With every origin allowed the answer is "*", and credentials are not
// mentioned. With AllowCredentials as well, as AllowAll is configured, the
// answer names the origin of the request instead: any origin may then send
// credentialed requests. The README warns about that combination.
func TestCorsWildcardAndCredentials(t *testing.T) {
	h := func(ctx iris.Context) { ctx.WriteString("ok") }

	app := iris.New()
	app.UseRouter(cors.Default())
	app.Get("/", h)

	r := httptest.New(t, app).GET("/").WithHeader("Origin", "https://anything.example").
		Expect().Status(httptest.StatusOK)
	r.Header("Access-Control-Allow-Origin").IsEqual("*")
	r.Headers().NotContainsKey("Access-Control-Allow-Credentials")

	app = iris.New()
	app.UseRouter(cors.AllowAll())
	app.Get("/", h)

	r = httptest.New(t, app).GET("/").WithHeader("Origin", "https://anything.example").
		Expect().Status(httptest.StatusOK)
	r.Header("Access-Control-Allow-Origin").IsEqual("https://anything.example")
	r.Header("Access-Control-Allow-Credentials").IsEqual("true")
}
