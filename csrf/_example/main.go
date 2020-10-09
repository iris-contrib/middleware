// This middleware provides Cross-Site Request Forgery
// protection.
//
// It securely generates a masked (unique-per-request) token that
// can be embedded in the HTTP response (e.g. form field or HTTP header).
// The original (unmasked) token is stored in the session, which is inaccessible
// by an attacker (provided you are using HTTPS). Subsequent requests are
// expected to include this token, which is compared against the session token.
// Requests that do not provide a matching token are served with a HTTP 403
// 'Forbidden' error response.
package main

import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/csrf"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	CSRF := csrf.Protect(
		// Note that the authentication key provided should be 32 bytes
		// long and persist across application restarts.
		[]byte("9AB0F421E53A477C084477AEA06096F5"),
		// WARNING: Set it to true on production with HTTPS.
		csrf.Secure(false),
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

	userAPI := app.Party("/user")
	userAPI.Use(CSRF)
	// To run this middleware on HTTP errors too:
	// UseError(protect)
	//
	// OR to run it everywhere(all child parties, subdomains, errors),
	// before the router itself:
	// UseRouter(protect)

	userAPI.Get("/signup", getSignupForm)
	// POST requests without a valid token will return a HTTP 403 Forbidden.
	userAPI.Post("/signup", postSignupForm)

	// Remove the CSRF middleware (1)
	userAPI.Post("/unprotected", unprotected).
		RemoveHandler(CSRF) // or RemoveHandler("iris-contrib.csrf.token")

		/* Skip the CSRF check for a Party (2)
		app.Use(func(ctx iris.Context){
			shouldSkipCSRF = [custom condition...]
			if shouldSkipCSRF {
				csrf.UnsafeSkipCheck(ctx)
			}
			ctx.Next()
		})
		app.Use(CSRF)
		*/

	// GET:  http://localhost:8080/user/signup
	// POST: http://localhost:8080/user/signup
	// POST:  http://localhost:8080/user/unprotected
	app.Listen(":8080")
}

func getSignupForm(ctx iris.Context) {
	// views/user/signup.html just needs a {{ .csrfField }} template tag for
	// csrf.TemplateField to inject the CSRF token into. Easy!
	ctx.ViewData(csrf.TemplateTag, csrf.TemplateField(ctx))
	ctx.View("user/signup.html")

	// We could also retrieve the token directly from csrf.Token(r) and
	// set it in the request header - ctx.GetHeader("X-CSRF-Token", token)
	// This is useful if you're sending JSON to clients or a front-end JavaScript
	// framework.
}

func postSignupForm(ctx iris.Context) {
	ctx.Writef("You're welcome mate!")
}

func unprotected(ctx iris.Context) {
	ctx.Writef("Hey, I am open to CSRF attacks!")
}
