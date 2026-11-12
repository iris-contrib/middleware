package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"

	"github.com/golang-jwt/jwt/v4"
)

var testSecret = []byte("My Secret")

func newTestMiddleware(cfg Config) *Middleware {
	cfg.ValidationKeyGetter = func(token *Token) (interface{}, error) {
		return testSecret, nil
	}
	cfg.SigningMethod = SigningMethodHS256

	return New(cfg)
}

func signTestToken(t *testing.T, method jwt.SigningMethod, claims MapClaims) string {
	t.Helper()

	s, err := NewTokenWithClaims(method, claims).SignedString(testSecret)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

// The default error handler answers 401 with the error's text and
// leaves the error on the Context for the handlers that run after it.
func TestOnError(t *testing.T) {
	j := newTestMiddleware(Config{})

	var stored error

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.Next()
		stored = ctx.GetErr()
	})
	app.Get("/secured", j.Serve, func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	e.GET("/secured").Expect().Status(iris.StatusUnauthorized).
		Body().IsEqual(ErrTokenMissing.Error())
	if !errors.Is(stored, ErrTokenMissing) {
		t.Fatalf("expected %v on the Context, got %v", ErrTokenMissing, stored)
	}

	e.GET("/secured").WithHeader("Authorization", "Basic abc").
		Expect().Status(iris.StatusUnauthorized).
		Body().IsEqual("authorization header format must be Bearer {token}")

	// A token signed with another algorithm than the configured one.
	other := signTestToken(t, SigningMethodHS512, MapClaims{"foo": "bar"})
	e.GET("/secured").WithHeader("Authorization", "Bearer "+other).
		Expect().Status(iris.StatusUnauthorized).
		Body().IsEqual("expected HS256 signing method but token specified HS512")

	valid := signTestToken(t, SigningMethodHS256, MapClaims{"foo": "bar"})
	e.GET("/secured").WithHeader("Authorization", "Bearer "+valid).
		Expect().Status(iris.StatusOK).Body().IsEqual("ok")
}

func TestExtractors(t *testing.T) {
	valid := signTestToken(t, SigningMethodHS256, MapClaims{"foo": "bar"})

	ok := func(ctx iris.Context) { ctx.WriteString("ok") }

	byQuery := newTestMiddleware(Config{Extractor: FromParameter("token")})
	byEither := newTestMiddleware(Config{Extractor: FromFirst(FromAuthHeader, FromParameter("token"))})
	optional := newTestMiddleware(Config{CredentialsOptional: true})

	app := iris.New()
	app.Get("/query", byQuery.Serve, ok)
	app.Get("/either", byEither.Serve, ok)
	app.Get("/optional", optional.Serve, func(ctx iris.Context) {
		if optional.Get(ctx) == nil {
			ctx.WriteString("anonymous")
			return
		}
		ctx.WriteString("authenticated")
	})

	e := httptest.New(t, app)

	// FromParameter reads the query only, the default extractor the header only.
	e.GET("/query").WithQuery("token", valid).Expect().Status(iris.StatusOK)
	e.GET("/query").WithHeader("Authorization", "Bearer "+valid).Expect().Status(iris.StatusUnauthorized)

	e.GET("/either").WithQuery("token", valid).Expect().Status(iris.StatusOK)
	e.GET("/either").WithHeader("Authorization", "Bearer "+valid).Expect().Status(iris.StatusOK)
	e.GET("/either").Expect().Status(iris.StatusUnauthorized)

	e.GET("/optional").Expect().Status(iris.StatusOK).Body().IsEqual("anonymous")
	e.GET("/optional").WithQuery("token", valid).Expect().Status(iris.StatusOK).Body().IsEqual("anonymous")
	e.GET("/optional").WithHeader("Authorization", "Bearer "+valid).
		Expect().Status(iris.StatusOK).Body().IsEqual("authenticated")
	// Optional credentials that are present must still be valid.
	e.GET("/optional").WithHeader("Authorization", "Bearer "+valid+"x").
		Expect().Status(iris.StatusUnauthorized)
}

func TestExpiration(t *testing.T) {
	j := newTestMiddleware(Config{Expiration: true})

	app := iris.New()
	app.Get("/secured", j.Serve, func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	fresh := signTestToken(t, SigningMethodHS256, MapClaims{"exp": time.Now().Add(time.Hour).Unix()})
	e.GET("/secured").WithHeader("Authorization", "Bearer "+fresh).Expect().Status(iris.StatusOK)

	expired := signTestToken(t, SigningMethodHS256, MapClaims{"exp": time.Now().Add(-time.Hour).Unix()})
	e.GET("/secured").WithHeader("Authorization", "Bearer "+expired).Expect().Status(iris.StatusUnauthorized)

	// An expired token is refused without the Expiration option too.
	lax := newTestMiddleware(Config{})
	laxApp := iris.New()
	laxApp.Get("/secured", lax.Serve, func(ctx iris.Context) { ctx.WriteString("ok") })
	laxE := httptest.New(t, laxApp)
	laxE.GET("/secured").WithHeader("Authorization", "Bearer "+expired).Expect().Status(iris.StatusUnauthorized)
	// What the option adds is the requirement that the claim exists.
	laxE.GET("/secured").WithHeader("Authorization", "Bearer "+signTestToken(t, SigningMethodHS256, MapClaims{"foo": "bar"})).
		Expect().Status(iris.StatusOK)

	// With Expiration set a token that carries no "exp" claim is refused.
	noExp := signTestToken(t, SigningMethodHS256, MapClaims{"foo": "bar"})
	e.GET("/secured").WithHeader("Authorization", "Bearer "+noExp).
		Expect().Status(iris.StatusUnauthorized).Body().IsEqual(ErrTokenExpired.Error())
}
