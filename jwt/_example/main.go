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
	token := JwtMiddleware.Get(ctx)

	ctx.Writef("This is an authenticated request\n")
	ctx.Writef("Claim content:\n")

	ctx.Writef("%s", token.Signature)

}

func main() {
	app := iris.New()
	app.Adapt(httprouter.New()) // adapt a router first of all

	JwtMiddleware = jwtmiddleware.New(jwtmiddleware.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte("My Secret"), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
		Extractor: jwtmiddleware.FromFirst(
			jwtmiddleware.FromAuthHeader,
			jwtmiddleware.FromParameter("auth_code")),
	})

	app.Get("/ping", jwtmiddleware.Serve, myHandler)
	app.Listen(":3001")
} // don't forget to look ../jwt_test.go to seee how to set your own custom claims
