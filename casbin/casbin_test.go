package casbin_test

import (
	"testing"

	"github.com/iris-contrib/middleware/casbin"

	casbinlib "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

const modelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

var policies = [][]string{
	{"alice", "/dataset1/*", "GET"},
	{"alice", "/dataset1/resource1", "POST"},
	{"bob", "/dataset2/resource1", "*"},
	{"bob", "/dataset2/resource2", "GET"},
	{"bob", "/dataset2/folder1/*", "POST"},
}

// newEnforcer builds an enforcer from a model and policies held in memory:
// no model file, no policy file, no adapter.
func newEnforcer(t *testing.T, text string) *casbinlib.Enforcer {
	t.Helper()

	m, err := model.NewModelFromString(text)
	if err != nil {
		t.Fatal(err)
	}

	e, err := casbinlib.NewEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = e.AddPolicies(policies); err != nil {
		t.Fatal(err)
	}

	return e
}

// subjectFromHeader stands in for an authentication middleware:
// it takes the subject from a request header.
func subjectFromHeader(ctx iris.Context) {
	casbin.SetSubject(ctx, ctx.Header("X-Subject"))
	ctx.Next()
}

func registerRoutes(app *iris.Application) {
	hi := func(ctx iris.Context) {
		ctx.Writef("Hello %s", casbin.Subject(ctx))
	}

	app.Any("/dataset1/{p:path}", hi)
	app.Any("/dataset2/resource1", hi)
	app.Get("/dataset2/resource2", hi)
	app.Post("/dataset2/folder1/{p:path}", hi)
}

type request struct {
	subject string
	method  string
	path    string
	status  int
}

func run(t *testing.T, app *iris.Application, tests []request) {
	t.Helper()

	e := httptest.New(t, app)
	for _, tt := range tests {
		e.Request(tt.method, tt.path).WithHeader("X-Subject", tt.subject).
			Expect().Status(tt.status)
	}
}

// allowedAndDenied are the requests whose answer does not depend on
// where the middleware is registered.
var allowedAndDenied = []request{
	{"alice", "GET", "/dataset1/resource1", httptest.StatusOK},
	{"alice", "POST", "/dataset1/resource1", httptest.StatusOK},
	{"alice", "GET", "/dataset1/resource2", httptest.StatusOK},
	{"alice", "POST", "/dataset1/resource2", httptest.StatusForbidden},
	{"alice", "GET", "/dataset2/resource1", httptest.StatusForbidden},
	{"bob", "GET", "/dataset2/resource1", httptest.StatusOK},
	{"bob", "DELETE", "/dataset2/resource1", httptest.StatusOK},
	{"bob", "GET", "/dataset2/resource2", httptest.StatusOK},
	{"bob", "POST", "/dataset2/folder1/item1", httptest.StatusOK},
	{"bob", "GET", "/dataset1/resource1", httptest.StatusForbidden},
	{"", "GET", "/dataset1/resource1", httptest.StatusForbidden},
	{"mallory", "GET", "/dataset1/resource1", httptest.StatusForbidden},
}

// Registered with UseRouter the middleware runs before the router,
// so a path with no route answers 403 unless a policy allows it.
func TestServeHTTPUseRouter(t *testing.T) {
	c := casbin.New(newEnforcer(t, modelText))

	app := iris.New()
	app.UseRouter(subjectFromHeader, c.ServeHTTP)
	registerRoutes(app)

	run(t, app, append(allowedAndDenied,
		// A policy allows the request but no route serves it.
		request{"bob", "POST", "/dataset2/folder1", httptest.StatusForbidden},
		request{"bob", "DELETE", "/dataset2/folder1/item1", httptest.StatusForbidden},
		request{"alice", "GET", "/nothing/here", httptest.StatusForbidden},
	))
}

// Use on the root application also runs on 404 and 405 in Iris v14,
// so it answers 403 for an unknown path like UseRouter does.
func TestServeHTTPUseOnRoot(t *testing.T) {
	c := casbin.New(newEnforcer(t, modelText))

	app := iris.New()
	app.Use(subjectFromHeader, c.ServeHTTP)
	registerRoutes(app)

	run(t, app, append(allowedAndDenied,
		request{"alice", "GET", "/nothing/here", httptest.StatusForbidden},
	))
}

// UseRelevant runs the middleware on matched routes only,
// so a path with no route answers 404.
func TestServeHTTPUseRelevant(t *testing.T) {
	c := casbin.New(newEnforcer(t, modelText))

	app := iris.New()
	app.UseRelevant(subjectFromHeader, c.ServeHTTP)
	registerRoutes(app)

	run(t, app, append(allowedAndDenied,
		request{"alice", "GET", "/nothing/here", httptest.StatusNotFound},
		request{"bob", "GET", "/dataset2/folder1/item1", httptest.StatusNotFound},
	))
}

// Without SetSubject the subject is the username of the Context's User,
// which is what an authentication middleware such as basicauth sets.
func TestSubjectFromUser(t *testing.T) {
	c := casbin.New(newEnforcer(t, modelText))

	app := iris.New()
	app.UseRelevant(func(ctx iris.Context) {
		if name := ctx.Header("X-User"); name != "" {
			if err := ctx.SetUser(iris.Map{"username": name}); err != nil {
				ctx.StopWithError(iris.StatusInternalServerError, err)
				return
			}
		}
		ctx.Next()
	}, c.ServeHTTP)
	registerRoutes(app)

	e := httptest.New(t, app)
	e.GET("/dataset1/resource1").WithHeader("X-User", "alice").
		Expect().Status(httptest.StatusOK).Body().IsEqual("Hello alice")
	e.GET("/dataset1/resource1").WithHeader("X-User", "bob").
		Expect().Status(httptest.StatusForbidden)
	// No user at all: the subject is empty and nothing allows it.
	e.GET("/dataset1/resource1").Expect().Status(httptest.StatusForbidden)
}

func TestSubjectExtractorAndUnauthorizedHandler(t *testing.T) {
	c := casbin.New(newEnforcer(t, modelText))
	c.SubjectExtractor = func(ctx iris.Context) string {
		return ctx.Query().Get("as")
	}
	c.UnauthorizedHandler = func(ctx iris.Context) {
		ctx.StopWithText(iris.StatusUnauthorized, "no access")
	}

	app := iris.New()
	app.UseRelevant(c.ServeHTTP)
	registerRoutes(app)

	e := httptest.New(t, app)
	e.GET("/dataset1/resource1").WithQuery("as", "alice").
		Expect().Status(httptest.StatusOK)
	e.GET("/dataset1/resource1").WithQuery("as", "bob").
		Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("no access")
}

// Check is an Iris Filter: it reports the decision and leaves the response alone.
func TestCheckAsFilter(t *testing.T) {
	c := casbin.New(newEnforcer(t, modelText))

	app := iris.New()
	app.Get("/dataset1/{p:path}", subjectFromHeader, func(ctx iris.Context) {
		if !c.Check(ctx) {
			ctx.StopWithText(iris.StatusTeapot, "denied by the handler")
			return
		}
		ctx.WriteString("allowed")
	})

	e := httptest.New(t, app)
	e.GET("/dataset1/resource1").WithHeader("X-Subject", "alice").
		Expect().Status(httptest.StatusOK).Body().IsEqual("allowed")
	e.GET("/dataset1/resource1").WithHeader("X-Subject", "bob").
		Expect().Status(iris.StatusTeapot).Body().IsEqual("denied by the handler")
}

// When the enforcer fails the request is denied and the reason is kept on the
// Context as a private error: readable by the server, not sent to the client.
func TestEnforcerErrorIsDeniedAndPrivate(t *testing.T) {
	// The matcher calls a function that does not exist, so Enforce returns an error.
	const broken = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = noSuchFunction(r.sub, p.sub)
`
	c := casbin.New(newEnforcer(t, broken))
	if _, err := c.Enforce(nilPathContext(t), "alice"); err == nil {
		t.Skip("this casbin version does not report an unknown matcher function as an error")
	}

	var stored error

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.Next()
		stored = ctx.GetErr()
	})
	app.UseRelevant(subjectFromHeader, c.ServeHTTP)
	registerRoutes(app)

	e := httptest.New(t, app)
	e.GET("/dataset1/resource1").WithHeader("X-Subject", "alice").
		Expect().Status(httptest.StatusForbidden).Body().IsEmpty()

	if stored == nil {
		t.Fatal("expected the enforcer error to be stored on the Context")
	}
}

// nilPathContext returns a Context good enough for Casbin.Enforce,
// which reads only the request's path and method.
func nilPathContext(t *testing.T) iris.Context {
	t.Helper()

	var got iris.Context
	app := iris.New()
	app.Get("/probe", func(ctx iris.Context) {
		got = ctx.Clone()
	})
	httptest.New(t, app).GET("/probe").Expect().Status(httptest.StatusOK)

	return got
}

func TestNewEnforcerFromFiles(t *testing.T) {
	if _, err := casbin.NewEnforcer("testdata/missing.conf", "testdata/missing.csv"); err == nil {
		t.Fatal("expected an error for model and policy files that do not exist")
	}

	c, err := casbin.NewEnforcer("_examples/router/casbinmodel.conf", "_examples/router/casbinpolicy.csv")
	if err != nil {
		t.Fatal(err)
	}

	app := iris.New()
	app.UseRelevant(subjectFromHeader, c.ServeHTTP)
	registerRoutes(app)

	e := httptest.New(t, app)
	e.GET("/dataset1/resource1").WithHeader("X-Subject", "alice").Expect().Status(httptest.StatusOK)
	e.GET("/dataset1/resource1").WithHeader("X-Subject", "bob").Expect().Status(httptest.StatusForbidden)
}
