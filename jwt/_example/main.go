package main

import (
	"time"

	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/jwt"
)

var mySecret = []byte("secret")

var j = jwt.New(jwt.Config{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return mySecret, nil
	},
	Expiration: true,
	// Extract by the "token" url.
	// There are plenty of options.
	// The default jwt's behavior to extract a token value is by
	// the `Authorization: Bearer $TOKEN` header.
	Extractor: jwt.FromParameter("token"),
	// When set, the middleware verifies that tokens are
	// signed with the specific signing algorithm
	// If the signing method is not constant the `jwt.Config.ValidationKeyGetter` callback
	// can be used to implement additional checks
	// Important to avoid security issues described here:
	// https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
	SigningMethod: jwt.SigningMethodHS256,
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
	tokenString, _ := token.SignedString(mySecret)

	ctx.HTML(`Token: ` + tokenString + `<br/><br/>
<a href="/protected?token=` + tokenString + `">/protected?token=` + tokenString + `</a>`)
}

func myAuthenticatedHandler(ctx iris.Context) {
	if err := j.CheckJWT(ctx); err != nil {
		j.Config.ErrorHandler(ctx, err)
		return
	}

	token := ctx.Values().Get("jwt").(*jwt.Token)

	ctx.Writef("This is an authenticated request\n\n")
	// ctx.Writef("Claim content:\n")

	foobar := token.Claims.(jwt.MapClaims)
	ctx.Writef("foo=%s\n", foobar["foo"])
	// for key, value := range foobar {
	// 	ctx.Writef("%s = %s", key, value)
	// }
}

func main() {
	app := iris.New()

	app.Get("/", getTokenHandler)
	app.Get("/protected", myAuthenticatedHandler)

	// j.CheckJWT(Context) error can be also used inside handlers.

	app.Listen(":8080")
}
