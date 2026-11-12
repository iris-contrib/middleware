package main

import (
	"time"

	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/jwt"
)

var mySecret = []byte("secret")

var j = jwt.New(jwt.Config{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return mySecret, nil
	},
	// Refuse a token that has expired or carries no "exp" claim.
	Expiration: true,
	// When set, the middleware verifies that tokens are
	// signed with the specific signing algorithm
	// If the signing method is not constant the `jwt.Config.ValidationKeyGetter` callback
	// can be used to implement additional checks
	// Important to avoid security issues described here:
	// https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
	SigningMethod: jwt.SigningMethodHS256,
	// The token is read from the `Authorization: Bearer $TOKEN` header,
	// which is the default. To read it from the URL query instead:
	//
	// Extractor: jwt.FromParameter("token"),
	//
	// Keep in mind that a token in a URL ends up in access logs,
	// in the browser history and in Referer headers.
})

// generate token to use.
func getTokenHandler(ctx iris.Context) {
	now := time.Now()
	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo": "bar",
		"iat": now.Unix(),
		"exp": now.Add(15 * time.Minute).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(mySecret)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, iris.PrivateError(err))
		return
	}

	ctx.WriteString(tokenString)
}

func myAuthenticatedHandler(ctx iris.Context) {
	// The token that j.Serve verified.
	token := j.Get(ctx)

	ctx.Writef("This is an authenticated request\n\n")

	foobar := token.Claims.(jwt.MapClaims)
	ctx.Writef("foo=%s\n", foobar["foo"])
}

func newApp() *iris.Application {
	app := iris.New()

	app.Get("/", getTokenHandler)
	// j.Serve stops the request with 401 when the token is missing or invalid.
	app.Get("/protected", j.Serve, myAuthenticatedHandler)

	// j.CheckJWT(ctx) error does the same check from inside a handler.

	return app
}

func main() {
	app := newApp()

	// $ TOKEN=$(curl -s http://localhost:8080)
	// $ curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/protected
	app.Listen(":8080")
}
