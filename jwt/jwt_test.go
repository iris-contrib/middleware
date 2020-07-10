package jwt

// Unlike the other middleware, this middleware was cloned from external source: https://github.com/auth0/go-jwt-middleware
// (because it used "context" to define the user but we don't need that so a simple iris.FromStd wouldn't work as expected.)
// jwt_test.go also didn't created by me:
// 28 Jul 2016
// @heralight heralight add jwt unit test.
//
// So if this doesn't works for you just try other net/http compatible middleware and bind it via `iris.FromStd(myHandlerWithNext)`,
// It's here for your learning curve.

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

type Response struct {
	Text string `json:"text"`
}

func TestBasicJwt(t *testing.T) {
	var (
		app = iris.New()
		j   = New(Config{
			ValidationKeyGetter: func(token *Token) (interface{}, error) {
				return []byte("My Secret"), nil
			},
			SigningMethod: SigningMethodHS256,
		})
	)

	securedPingHandler := func(ctx iris.Context) {
		userToken := j.Get(ctx)
		var claimTestedValue string
		if claims, ok := userToken.Claims.(MapClaims); ok && userToken.Valid {
			claimTestedValue = claims["foo"].(string)
		} else {
			claimTestedValue = "Claims Failed"
		}

		response := Response{"Iauthenticated" + claimTestedValue}
		// get the *Token which contains user information using:
		// user:= j.Get(ctx) or ctx.Values().Get("jwt").(*Token)

		ctx.JSON(response)
	}

	app.Get("/secured/ping", j.Serve, securedPingHandler)
	e := httptest.New(t, app)

	e.GET("/secured/ping").Expect().Status(iris.StatusUnauthorized)

	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := NewTokenWithClaims(SigningMethodHS256, MapClaims{
		"foo": "bar",
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString([]byte("My Secret"))

	e.GET("/secured/ping").WithHeader("Authorization", "Bearer "+tokenString).
		Expect().Status(iris.StatusOK).Body().Contains("Iauthenticated").Contains("bar")
}
