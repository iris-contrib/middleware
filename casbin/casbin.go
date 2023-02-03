package casbin

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"

	"github.com/casbin/casbin/v2"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/casbin.*", "iris-contrib.casbin")
}

// Casbin is the auth service which contains the casbin enforcer.
type Casbin struct {
	enforcer *casbin.Enforcer
	// SubjectExtractor is used to extract the
	// current request's subject for the casbin role enforcer.
	// Defaults to the `Subject` package-level function which
	// extracts the subject from a prior registered authorization middleware's
	// username (e.g. basicauth or JWT).
	SubjectExtractor func(iris.Context) string

	// UnauthorizedHandler sets a custom handler to be executed
	// when the role checks fail.
	// Defaults to a handler which sends a status forbidden (403) status code.
	UnauthorizedHandler iris.Handler
}

// New returns the Casbin middleware based on the given casbin.Enforcer instance.
// The authorization determines a request based on `{subject, object, action}`.
// Please refer to: https://github.com/casbin/casbin to understand how it works first.
//
// The object is the current request's path and the action is the current request's method.
// The subject that casbin requires is extracted by:
//   - SubjectExtractor
//   - casbin.Subject
//     | set with casbin.SetSubject
//   - Context.User().GetUsername()
//     | by a prior auth middleware through Context.SetUser.
func New(e *casbin.Enforcer) *Casbin {
	return &Casbin{
		enforcer: e,
		SubjectExtractor: func(ctx iris.Context) string {
			return Subject(ctx)
		},
		UnauthorizedHandler: func(ctx iris.Context) {
			ctx.StopWithStatus(iris.StatusForbidden)
		},
	}
}

// NewEnforcer returns the Casbin middleware based on the given model and policy file paths.
//
// Read `New` package-level function for more information.
func NewEnforcer(modelFile, policyFile string) (*Casbin, error) {
	e, err := casbin.NewEnforcer(modelFile, policyFile)
	if err != nil {
		return nil, err
	}

	return New(e), nil
}

// ServeHTTP is the iris compatible casbin handler which should be passed to specific routes or parties.
// Responds with Status Forbidden on unauthorized clients.
// Usage:
// - app.Use(authMiddleware)
// - app.Use(casbinMiddleware.ServeHTTP) OR
// - app.UseRouter(casbinMiddleware.ServeHTTP) OR per route:
// - app.Get("/dataset1/resource1", casbinMiddleware.ServeHTTP, myHandler)
func (c *Casbin) ServeHTTP(ctx iris.Context) {
	if !c.Check(ctx) {
		c.UnauthorizedHandler(ctx)
		return
	}

	ctx.Next()
}

// Check checks the username, request's method and path and
// returns true if permission grandted otherwise false.
//
// It's an Iris Filter.
// Usage:
// - inside a handler
// - using the iris.NewConditionalHandler
func (c *Casbin) Check(ctx iris.Context) bool {
	ok, _ := c.Enforce(ctx, c.SubjectExtractor(ctx))
	return ok
}

// Enforce accepts the Context's path and method and a subject/role/username
// and reports whether the specific "subject" has access to the current request.
func (c *Casbin) Enforce(ctx iris.Context, subject string) (bool, error) {
	return c.enforcer.Enforce(subject, ctx.Path(), ctx.Method())
}

const subjectContextKey = "iris.contrib.casbin.subject"

// SetSubject sets a custom subject for the current request for the casbin middleware.
// See `Subject` package-level function too.
func SetSubject(ctx iris.Context, subject string) {
	ctx.Values().Set(subjectContextKey, subject)
}

// Subject gets the subject from an authorization middleware's username, e.g. basicauth or JWT.
// If there is no registered middleware to fetch the subject then
// it tries to extract it from the context's values (see SetSubject package-level function to set it).
func Subject(ctx iris.Context) string {
	subject := ctx.Values().GetString(subjectContextKey)
	if subject == "" {
		if u := ctx.User(); u != nil {
			subject, _ = u.GetUsername()
		}
	}
	return subject
}
