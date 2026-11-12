package casbin

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"

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

// ServeHTTP is the Iris handler of the casbin middleware.
// It answers 403 (Forbidden) through the UnauthorizedHandler when the subject
// is not allowed to access the request's path with the request's method.
//
// An authentication middleware that sets the subject must run before it.
// Where the pair is registered decides what an unknown path answers:
//
//   - app.UseRouter(auth, casbinMiddleware.ServeHTTP) runs before the router,
//     on every request. A path with no route answers 403 unless a policy allows it.
//   - app.Use(auth, casbinMiddleware.ServeHTTP) on the root application behaves
//     the same for unknown paths, because Use on the root also runs on 404 and 405.
//   - app.UseRelevant(auth, casbinMiddleware.ServeHTTP), Use on a child router,
//     or the handler passed to a single route run on matched routes only.
//     A path with no route answers 404.
func (c *Casbin) ServeHTTP(ctx iris.Context) {
	if !c.Check(ctx) {
		c.UnauthorizedHandler(ctx)
		return
	}

	ctx.Next()
}

// Check checks the subject, the request's method and path and
// returns true if permission is granted, otherwise false.
//
// When the enforcer itself fails (a matcher that does not evaluate, an adapter
// error) the request is denied and the error is stored on the Context as a
// private one: ctx.GetErr returns it for logging, and no error handler writes
// it to the client.
//
// It's an Iris Filter.
// Usage:
// - inside a handler
// - using the iris.NewConditionalHandler
func (c *Casbin) Check(ctx iris.Context) bool {
	ok, err := c.Enforce(ctx, c.SubjectExtractor(ctx))
	if err != nil {
		ctx.SetErr(iris.PrivateError(err))
		return false
	}

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
