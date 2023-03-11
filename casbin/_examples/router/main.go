package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"

	"github.com/iris-contrib/middleware/casbin"
)

// $ go get github.com/casbin/casbin/v2@v2.65.1
// $ go run main.go

func newApp() *iris.Application {
	app := iris.New()

	casbinMiddleware, err := casbin.NewEnforcer("casbinmodel.conf", "casbinpolicy.csv")
	if err != nil {
		panic(err)
	}
	/* The Casbin authorization determines a request based on `{subject, object, action}`.
	Please refer to: https://github.com/casbin/casbin to understand how it works first.
	The object is the current request's path and the action is the current request's method.
	The subject is extracted by the current request's ctx.User().GetUsername(),
	you can customize it by:
		1. casbinMiddleware.SubjectExtractor = func(ctx iris.Context) string {
			// [...custom logic]
			return "bob"
		}
		2. by SetSubject package-level function:
			func auth(ctx iris.Context) {
				casbin.SetSubject(ctx, "bob")
				ctx.Next()
			}
	*/
	app.UseRouter(basicauth.Default(map[string]string{
		"bob":   "bobpass",
		"alice": "alicepass",
	}))

	// Note that by registering with UseRouter instead of Use,
	// and becauese the middleware stops the execution with 403 (Forbidden) by default,
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
	ctx.Writef("Hello %s", casbin.Subject(ctx))
	// Note that, by default, the username is extracted by ctx.User().GetUsername()
	// to change that behavior modify the `casbin.SubjectExtractor` or
	// use the `casbin.SetSubject` to set a custom subject for the current request
	// before the casbin middleware's execution.
}
