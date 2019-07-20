package main

import (
	"github.com/kataras/iris"

	"github.com/iris-contrib/middleware/jwt"
)

var mySecret = []byte("My Secret")

// generate token to use.
func getTokenHandler(ctx iris.Context) {
	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo": "bar",
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString(mySecret)

	ctx.HTML(`Token: ` + tokenString + `<br/><br/>
<a href="/secured?token=` + tokenString + `">/secured?token=` + tokenString + `</a>`)
}

func myAuthenticatedHandler(ctx iris.Context) {
	user := ctx.Values().Get("jwt").(*jwt.Token)

	ctx.Writef("This is an authenticated request\n")
	ctx.Writef("Claim content:\n")

	foobar := user.Claims.(jwt.MapClaims)
	for key, value := range foobar {
		ctx.Writef("%s = %s", key, value)
	}
}

func main() {
	app := iris.New()

	j := jwt.New(jwt.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return mySecret, nil
		},

		// Extract by the "token" url.
		// There are plenty of options.
		// The default jwt's behavior to extract a token value is by
		// the `Authentication: Bearer $TOKEN` header.
		Extractor: jwt.FromParameter("token"),
		// When set, the middleware verifies that tokens are
		// signed with the specific signing algorithm
		// If the signing method is not constant the `jwt.Config.ValidationKeyGetter` callback
		// can be used to implement additional checks
		// Important to avoid security issues described here:
		// https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
		SigningMethod: jwt.SigningMethodHS256,
	})

	app.Get("/", getTokenHandler)
	app.Get("/secured", j.Serve, myAuthenticatedHandler)

	// j.CheckJWT(Context) error can be also used inside handlers.

	app.Run(iris.Addr(":8080"))
}
