package casbin

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"

	"github.com/casbin/casbin/v2"
)

/*
	Updated for the casbin 2.9 released 2 days ago.
	2020-08-08
*/

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/casbin.*", "iris-contrib.casbin")
}

// New returns the auth service which receives a casbin enforcer.
//
// Adapt with its `Wrapper` for the entire application
// or its `ServeHTTP` for specific routes or parties.
func New(e *casbin.Enforcer) *Casbin {
	return &Casbin{enforcer: e}
}

// ServeHTTP is the iris compatible casbin handler which should be passed to specific routes or parties.
// Usage:
// - app.Get("/dataset1/resource1", casbinMiddleware.ServeHTTP, myHandler)
// - app.Use(casbinMiddleware.ServeHTTP)
// - app.UseRouter(casbinMiddleware.ServeHTTP)
func (c *Casbin) ServeHTTP(ctx iris.Context) {
	if !c.Check(ctx) {
		ctx.StopWithStatus(iris.StatusForbidden)
		return
	}

	ctx.Next()
}

// Casbin is the auth services which contains the casbin enforcer.
type Casbin struct {
	enforcer *casbin.Enforcer
}

// Check checks the username, request's method and path and
// returns true if permission grandted otherwise false.
//
// It's an Iris Filter.
// Usage:
// - inside a handler
// - using the iris.NewConditionalHandler
func (c *Casbin) Check(ctx iris.Context) bool {
	username := Username(ctx)
	ok, _ := c.enforcer.Enforce(username, ctx.Path(), ctx.Method())
	return ok
}

const usernameContextKey = "iris.contrib.casbin.username"

// Username gets the username from the basicauth
// or the given (by a prior middleware) username.
// See `SetUsername` package-level function too.
func Username(ctx iris.Context) string {
	username := ctx.Values().GetString(usernameContextKey)
	if username == "" {
		username, _, _ = ctx.Request().BasicAuth()
	}

	return username
}

// SetUsername sets a custom username for the casbin middleware.
// See `Username` package-level function too.
func SetUsername(ctx iris.Context, username string) {
	ctx.Values().Set(usernameContextKey, username)
}
