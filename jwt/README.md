# jwt

JSON Web Token verification for [Iris](https://github.com/kataras/iris) v14, built on [golang-jwt/jwt](https://github.com/golang-jwt/jwt) v4.

**Iris has its own JWT middleware**, [kataras/iris/middleware/jwt](https://github.com/kataras/iris/tree/main/middleware/jwt). It signs as well as verifies, decodes claims into your own struct, supports encrypted payloads and a blocklist, and is maintained with the framework. Prefer it for new code. This module remains for applications that already use it.

## Install

```sh
go get github.com/iris-contrib/middleware/jwt@main
```

```go
import "github.com/iris-contrib/middleware/jwt"
```

## Example

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/jwt"
)

// In production load the secret from the environment or from a file.
var mySecret = []byte("My Secret")

func main() {
	app := iris.New()

	j := jwt.New(jwt.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return mySecret, nil
		},
		// Refuse a token signed with any other algorithm.
		SigningMethod: jwt.SigningMethodHS256,
		// Refuse a token that has expired or has no "exp" claim.
		Expiration: true,
	})

	// Issues a token.
	app.Get("/token", func(ctx iris.Context) {
		token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"foo": "bar",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		})

		tokenString, err := token.SignedString(mySecret)
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, iris.PrivateError(err))
			return
		}

		ctx.WriteString(tokenString)
	})

	// j.Serve answers 401 unless the request carries a valid token in the
	// "Authorization: Bearer <token>" header.
	app.Get("/protected", j.Serve, func(ctx iris.Context) {
		// The token that j.Serve verified.
		token := j.Get(ctx)

		claims := token.Claims.(jwt.MapClaims)
		ctx.Writef("foo=%s", claims["foo"])
	})

	// To protect a group of routes: app.Router("/users", j.Serve)

	app.Listen(":8080")
}
```

```sh
TOKEN=$(curl -s http://localhost:8080/token)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/protected
```

`j.CheckJWT(ctx)` runs the same check from inside a handler and returns the error instead of answering.

## Config

| Field | Default | Description |
| --- | --- | --- |
| `ValidationKeyGetter` | none, required | Returns the key that verifies the signature: a shared secret or a public key. |
| `SigningMethod` | none | When set, a token signed with another algorithm is refused. Set it. Without it the algorithm named in the token is trusted. |
| `Expiration` | false | When true, a token without an `exp` claim is refused. An `exp` in the past is always refused. |
| `Extractor` | `FromAuthHeader` | Where the token is read from. |
| `ContextKey` | `"jwt"` | The Context key the verified `*jwt.Token` is stored under. `Middleware.Get(ctx)` reads it. |
| `ErrorHandler` | `OnError` | Called with the error of a refused request. |
| `CredentialsOptional` | false | When true, a request without a token passes and `Get(ctx)` returns nil. A token that is present must still be valid. |
| `EnableAuthOnOptions` | false | When false, OPTIONS requests pass unchecked, for CORS preflight. |

`OnError` answers `401 Unauthorized` with the error's text and stores the error on the Context, where `ctx.GetErr()` returns it.

## Token extractors

| Extractor | Reads the token from |
| --- | --- |
| `FromAuthHeader` | The `Authorization: Bearer <token>` request header. |
| `FromParameter(name)` | The URL query parameter `name`. |
| `FromFirst(extractors...)` | The first of the given extractors that finds one. |

A token in a URL is written to access logs, kept in the browser history and sent in `Referer` headers. Use `FromParameter` only where a header cannot be set, such as a download link. The framework's own middleware stopped reading the URL by default in v14 for this reason.

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/jwt"
)

func main() {
	app := iris.New()

	j := jwt.New(jwt.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte("My Secret"), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
		// The header first, then the "token" URL query parameter.
		Extractor: jwt.FromFirst(jwt.FromAuthHeader, jwt.FromParameter("token")),
	})

	app.Get("/download", j.Serve, func(ctx iris.Context) {
		ctx.WriteString("the file")
	})

	app.Listen(":8080")
}
```

A `TokenExtractor` is a `func(iris.Context) (string, error)`. Return an empty string and no error when the request has no token.

## Names re-exported from golang-jwt

`Token`, `MapClaims`, `Claims`, `NewToken`, `NewTokenWithClaims` and the signing methods `SigningMethodHS256`, `HS384`, `HS512`, `ES256`, `ES384` and `ES512`. For anything else import `github.com/golang-jwt/jwt/v4` directly.
