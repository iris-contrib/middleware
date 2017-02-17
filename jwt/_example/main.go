// Iris provides some basic middleware, most for your learning courve.
// You can use any net/http compatible middleware with iris.ToHandler wrapper.
//
// JWT net/http video tutorial for golang newcomers: https://www.youtube.com/watch?v=dgJFeqeXVKw
//
// This middleware is the only one cloned from external source: https://github.com/auth0/go-jwt-middleware
// (because it used "context" to define the user but we don't need that so a simple iris.ToHandler wouldn't work as expected.)
package main

import (
	"github.com/dgrijalva/jwt-go"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func myHandler(ctx *iris.Context) {
	user := ctx.Get("user").(*jwt.Token)

	ctx.Writef("This is an authenticated request\n")
	ctx.Writef("Claim content:\n")

	ctx.Writef("%s", user.Signature)

}

func main() {
	app := iris.New()
	app.Adapt(httprouter.New()) // adapt a router first of all

	jwtHandler := jwtmiddleware.New(jwtmiddleware.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte("My Secret"), nil
		},
		// When set, the middleware verifies that tokens are signed with the specific signing algorithm
		// If the signing method is not constant the ValidationKeyGetter callback can be used to implement additional checks
		// Important to avoid security issues described here: https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
		SigningMethod: jwt.SigningMethodHS256,
	})

	app.Use(jwtHandler) // or .Get("/ping", jwtHandler.Serve, myHandler)

	app.Get("/ping", myHandler)
	app.Listen(":3001")
} // don't forget to look ../jwt_test.go to seee how to set your own custom claims
