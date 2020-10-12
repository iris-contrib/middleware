package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"

	"github.com/casbin/casbin/v2"
	cm "github.com/iris-contrib/middleware/casbin"
)

// $ go get github.com/casbin/casbin/v2@v2.13.1
// $ go run main.go

// Enforcer maps the model and the policy for the casbin service, we use this variable on the main_test too.
var Enforcer, _ = casbin.NewEnforcer("casbinmodel.conf", "casbinpolicy.csv")

func newApp() *iris.Application {
	app := iris.New()

	casbinMiddleware := cm.New(Enforcer)
	/* Casbin requires an authenticated user name,
	   You have three ways to set that username:
	1. casbinMiddleware.UsernameExtractor = func(ctx iris.Context) string {
		// [...custom logic]
		return "bob"
	}
	2. by SetUsername package-level function:
		func auth(ctx iris.Context) {
			cm.SetUsername(ctx, "bob")
			ctx.Next()
		}
	3. By registering an auth middleware that fills the Context.User()
	   ^ recommended way, and that's what it's used on that example.
	*/
	app.UseRouter(basicauth.Default(map[string]string{
		"bob":   "bobpass",
		"alice": "alicepass",
	}))
	// Note that by registering with UseRouter,
	// and becauese the middleware stops the execution with 403 (Forbidden)
	// if the authentication and roles match failed,
	// unregistered route paths will fire 403 instead of 404 (Not Found).
	app.UseRouter(casbinMiddleware.ServeHTTP)

	app.Get("/", hi)

	app.Any("/dataset1/{p:path}", hi) // p, dataset1_admin, /dataset1/*, * && p, alice, /dataset1/*, GET

	app.Post("/dataset1/resource1", hi)

	app.Get("/dataset2/resource2", hi)
	app.Post("/dataset2/folder1/{p:path}", hi)

	app.Any("/dataset2/resource1", hi)

	return app
}

func main() {
	app := newApp()
	app.Listen(":8080")
}

func hi(ctx iris.Context) {
	ctx.Writef("Hello %s", cm.Username(ctx))
	// Note that, by default, the username is extracted by ctx.Request().BasicAuth
	// to change that, use the `cm.SetUsername` before the casbin middleware's execution.
}
