// This example protects a signup form against Cross-Site Request Forgery.
//
// The middleware generates a masked (unique-per-request) token that
// can be embedded in the HTTP response (e.g. form field or HTTP header).
// The original (unmasked) token is stored in the session, which is inaccessible
// by an attacker (provided you are using HTTPS). Subsequent requests are
// expected to include this token, which is compared against the session token.
// Requests that do not provide a matching token are served with a HTTP 403
// 'Forbidden' error response.
//
// This directory is a Go module of its own, because it needs a template
// engine that the middleware itself does not depend on. Run it with:
//
//	$ go run .
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/view"

	"github.com/kataras/blocks"

	"github.com/iris-contrib/middleware/csrf"
)

func main() {
	app := newApp()

	// GET:  http://localhost:8080/user/signup
	// POST: http://localhost:8080/user/signup
	// POST: http://localhost:8080/user/unprotected
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	templates := blocks.New("./views").Reload(true)
	if err := templates.Load(); err != nil {
		panic(err)
	}
	app.UseRouter(view.Handler(templates))

	CSRF := csrf.Protect(
		// Note that the authentication key provided should be 32 bytes
		// long and persist across application restarts.
		[]byte("9AB0F421E53A477C084477AEA06096F5"),
		// WARNING: Set it to true on production with HTTPS.
		csrf.Secure(false),
		// you can set custom cookie name:
		// csrf.CookieName("csrfToken"),
	)

	/*
		Further customizations with the New package-level function:
		CSRF := csrf.New(csrf.Options{
			RequestHeader: "X-CSRF-Token",
			FieldName:     "csrf.token",
			ErrorHandler:  csrf.UnauthorizedHandler,
			Store: csrf.NewCookieStore(
				[]byte("9AB0F421E53A477C084477AEA06096F5"), csrf.Secure(false)),
		})

		CSRF.Filter  - is an iris Filter: func(iris.Context) bool
		CSRF.Protect - is an iris Handler, the middleware: func(iris.Context)
	*/

	// Use on a child router runs the middleware on the matched routes of
	// that router. To run it on every request, before the router itself:
	// app.UseRouter(CSRF)
	userAPI := app.Router("/user")
	userAPI.Use(CSRF)

	userAPI.Get("/signup", getSignupForm)
	// POST requests without a valid token will return a HTTP 403 Forbidden.
	userAPI.Post("/signup", postSignupForm)

	// A route that must skip the CSRF check is registered outside the
	// protected router (1).
	app.Post("/user/unprotected", unprotected)

	/* Or skip the check by a condition, with a handler that runs before
	the CSRF one (2):

	userAPI.Use(func(ctx iris.Context) {
		if shouldSkipCSRF(ctx) {
			csrf.UnsafeSkipCheck(ctx)
		}
		ctx.Next()
	})
	userAPI.Use(CSRF)
	*/

	return app
}

func getSignupForm(ctx iris.Context) {
	// views/user/signup.html just needs a {{ .csrfField }} template tag for
	// csrf.TemplateField to inject the CSRF token into.
	view.KeyValue(ctx, csrf.TemplateTag, csrf.TemplateField(ctx))
	view.Render(ctx, "user/signup", "", nil)

	// We could also retrieve the token directly from csrf.Token(ctx) and
	// set it in the response header: ctx.SetHeader("X-CSRF-Token", token).
	// This is useful if you're sending JSON to clients or a front-end JavaScript
	// framework.
}

func postSignupForm(ctx iris.Context) {
	ctx.Writef("You're welcome mate!")
}

func unprotected(ctx iris.Context) {
	ctx.Writef("Hey, I am open to CSRF attacks!")
}
