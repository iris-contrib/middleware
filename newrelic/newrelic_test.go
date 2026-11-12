package newrelic_test

import (
	"testing"

	nriris "github.com/iris-contrib/middleware/newrelic"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

// newTestApp returns a New Relic application that never connects: the agent
// keeps what it would have sent in memory, where the Expect methods read it.
func newTestApp() integrationsupport.ExpectApp {
	return integrationsupport.NewBasicTestApp()
}

func expectWebTransaction(t *testing.T, app integrationsupport.ExpectApp, name string) {
	t.Helper()

	app.ExpectMetricsPresent(t, []integrationsupport.WantMetric{
		{Name: "WebTransaction/Go/" + name, Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
	})
}

func hello(ctx iris.Context) {
	ctx.SetHeader("X-Handler", "hello")
	ctx.WriteString("hello")
}

// The transaction is named after the method and the handler that follows
// the middleware, and the response reaches the client untouched.
func TestMiddleware(t *testing.T) {
	nrApp := newTestApp()

	var inHandler string

	app := iris.New()
	app.Use(nriris.Middleware(nrApp.Application))
	app.Get("/hello", func(ctx iris.Context) {
		if txn := nriris.Transaction(ctx); txn != nil {
			inHandler = txn.Name()
		}
		ctx.Next()
	}, hello)

	e := httptest.New(t, app)
	r := e.GET("/hello").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("hello")
	r.Header("X-Handler").IsEqual("hello")

	if inHandler == "" {
		t.Fatal("expected the handler to find the transaction on the Context")
	}

	expectWebTransaction(t, nrApp, inHandler)
}

func TestTransactionName(t *testing.T) {
	nrApp := newTestApp()

	app := iris.New()
	app.Get("/hello", nriris.New(nrApp.Application), hello)

	httptest.New(t, app).GET("/hello").Expect().Status(httptest.StatusOK)

	expectWebTransaction(t, nrApp, "GET "+context.HandlerName(hello))
}

// MiddlewareWithFullPath names the transaction after the route as registered.
func TestMiddlewareWithFullPath(t *testing.T) {
	for _, mw := range []func(*newrelic.Application) iris.Handler{
		nriris.MiddlewareWithFullPath,
		nriris.MiddleareWithFullPath, // the deprecated, misspelled name.
	} {
		nrApp := newTestApp()

		app := iris.New()
		app.Use(mw(nrApp.Application))
		app.Get("/users/{id:int}", func(ctx iris.Context) {
			ctx.Writef("user %d", ctx.Params().GetInt("id"))
		})

		e := httptest.New(t, app)
		e.GET("/users/42").Expect().Status(httptest.StatusOK).Body().IsEqual("user 42")
		e.GET("/users/7").Expect().Status(httptest.StatusOK).Body().IsEqual("user 7")

		// One name for every id.
		expectWebTransaction(t, nrApp, "GET /users/{id:int}")
	}
}

// The handler reaches the transaction through the Context, here to rename it.
func TestTransactionFromContext(t *testing.T) {
	nrApp := newTestApp()

	app := iris.New()
	app.Use(nriris.Middleware(nrApp.Application))
	app.Get("/", func(ctx iris.Context) {
		txn := nriris.GetTransaction(ctx)
		if txn == nil {
			ctx.StopWithText(iris.StatusInternalServerError, "no transaction")
			return
		}

		txn.SetName("custom-name")
		ctx.WriteString("renamed")
	})

	httptest.New(t, app).GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("renamed")

	expectWebTransaction(t, nrApp, "custom-name")
}

// The status code the handler set is what the agent records:
// a 500 is reported as an error of the transaction.
func TestStatusCodeIsReported(t *testing.T) {
	nrApp := newTestApp()

	app := iris.New()
	app.Use(nriris.MiddlewareWithFullPath(nrApp.Application))
	app.Get("/fail", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusInternalServerError)
	})
	app.Get("/fail-with-body", func(ctx iris.Context) {
		ctx.StopWithText(iris.StatusInternalServerError, "it failed")
	})

	e := httptest.New(t, app)
	e.GET("/fail").Expect().Status(httptest.StatusInternalServerError).Body().IsEmpty()
	e.GET("/fail-with-body").Expect().Status(httptest.StatusInternalServerError).Body().IsEqual("it failed")

	nrApp.ExpectMetricsPresent(t, []integrationsupport.WantMetric{
		{Name: "Errors/WebTransaction/Go/GET /fail", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/WebTransaction/Go/GET /fail-with-body", Scope: "", Forced: true, Data: nil},
	})
}

// With Use on the root application a request that matches no route is
// instrumented too, and named after its method and full URI.
func TestNotFoundIsInstrumented(t *testing.T) {
	nrApp := newTestApp()

	var name string

	app := iris.New()
	app.Use(nriris.MiddlewareWithFullPath(nrApp.Application))
	app.HandleNoRoute(func(ctx iris.Context) {
		if txn := nriris.Transaction(ctx); txn != nil {
			name = txn.Name()
		}
		ctx.WriteString("nothing here")
	})

	httptest.New(t, app).GET("/missing").Expect().Status(httptest.StatusNotFound).
		Body().IsEqual("nothing here")

	if name == "" {
		t.Fatal("expected a transaction for a request that matched no route")
	}

	expectWebTransaction(t, nrApp, name)
}

// A nil application turns the middleware into a pass-through.
func TestNilApplication(t *testing.T) {
	app := iris.New()
	app.Use(nriris.Middleware(nil))
	app.Get("/", func(ctx iris.Context) {
		if nriris.Transaction(ctx) != nil {
			ctx.StopWithText(iris.StatusInternalServerError, "unexpected transaction")
			return
		}
		ctx.WriteString("ok")
	})

	httptest.New(t, app).GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
}

// The middleware replaces the Context's response writer while the chain runs
// and puts the previous one back when it returns.
func TestResponseWriterIsRestored(t *testing.T) {
	nrApp := newTestApp()

	var before, during, after context.ResponseWriter

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		before = ctx.ResponseWriter()
		ctx.Next()
		after = ctx.ResponseWriter()
	})
	app.Use(nriris.Middleware(nrApp.Application))
	app.Get("/", func(ctx iris.Context) {
		during = ctx.ResponseWriter()
		ctx.WriteString("ok")
	})

	httptest.New(t, app).GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")

	if during == before {
		t.Fatal("expected the handler to write through the middleware's response writer")
	}

	if after != before {
		t.Fatal("expected the previous response writer back on the Context after the chain")
	}
}
