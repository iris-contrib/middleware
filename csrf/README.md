# csrf

[Cross-site request forgery](https://owasp.org/www-community/attacks/csrf) protection for [Iris](https://github.com/kataras/iris) v14. A fork of [gorilla/csrf](https://github.com/gorilla/csrf) with a replaceable token [Store](store.go).

The middleware keeps a random token per browser, in a signed cookie by default. Every request gets a masked copy of it to embed in a form field or to send back in a header. A request with a method other than GET, HEAD, OPTIONS or TRACE must return that copy, or it is answered with `403 Forbidden`.

## Install

```sh
go get github.com/iris-contrib/middleware/csrf@main
```

```go
import "github.com/iris-contrib/middleware/csrf"
```

## Example

A JSON API whose client reads the token from a response header and sends it back in the `X-CSRF-Token` request header:

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/csrf"
)

func main() {
	app := iris.New()

	protect := csrf.Protect(
		// The authentication key must be 32 bytes long and stay the same
		// across restarts. A new key invalidates every cookie already issued.
		[]byte("9AB0F421E53A477C084477AEA06096F5"),
		// The cookie is Secure by default, so it is not sent over plain HTTP.
		// Turn that off for local development only.
		csrf.Secure(false),
	)

	api := app.Router("/api")
	api.Use(protect)

	// A GET request is never refused. It issues the cookie and the token.
	api.Get("/user", func(ctx iris.Context) {
		ctx.SetHeader("X-CSRF-Token", csrf.Token(ctx))
		ctx.JSON(iris.Map{"name": "kataras"})
	})

	// A POST without the X-CSRF-Token header (or the "csrf.token" form
	// field) and the cookie is answered with 403.
	api.Post("/user", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"saved": true})
	})

	app.Listen(":8080")
}
```

For an HTML form, pass `csrf.TemplateField(ctx)` to the template under the `csrf.TemplateTag` key (`csrfField`). It renders `<input type="hidden" name="csrf.token" value="...">`. The [_example](_example) directory is a complete form example. It is a Go module of its own because it needs a template engine, which the middleware does not depend on:

```sh
cd _example
go run .
```

## How a request is checked

1. `csrf.UnsafeSkipCheck(ctx)`, called by an earlier handler, skips everything below.
2. The token is read from the store. When there is none, a new one is generated and saved, which sets the cookie.
3. For GET, HEAD, OPTIONS and TRACE the check ends here.
4. On HTTPS the `Referer` header must be present and its host must be the request's host or one of `Options.TrustedOrigins`.
5. The token sent by the client is read from the request header first, then from the form field, and compared in constant time.

`Vary: Cookie` is added to every response that passes.

Reading the form field parses the request body as a form. A handler that needs the raw body of a form request must capture it in a middleware registered before this one.

Step 4 looks at the connection of the Go server itself. Behind a proxy that terminates TLS the server sees plain HTTP, and the Referer check does not run.

## Functions

| Name | Description |
| --- | --- |
| `Protect(authKey []byte, cookieOpts ...CookieOption) iris.Handler` | The middleware, with the cookie store. |
| `New(Options) *CSRF` | The same with every option. Panics without a `Store`. |
| `(*CSRF).Protect` | The handler: runs `Filter`, then the next handler or `Options.ErrorHandler`. |
| `(*CSRF).Filter(ctx) bool` | The check alone. It does not call the `ErrorHandler`. Works with `iris.NewConditionalHandler`. |
| `(*CSRF).RequestToken(ctx) []byte` | The token the client sent, decoded. |
| `Token(ctx) string` | The masked token of the current request. Empty when the middleware did not run. |
| `TemplateField(ctx) template.HTML` | The hidden `<input>` field with the token. |
| `FailureReason(ctx) error` | Why the request was refused. |
| `UnsafeSkipCheck(ctx)` | Skips the check for this request. Call it before the middleware runs, and only for requests that cannot be forged, such as an API client that authenticates with a token. |
| `UnauthorizedHandler` | The default `ErrorHandler`. |
| `NewCookieStore(authKey []byte, cookieOpts ...CookieOption) Store` | The default store. |

## Options

| Field | Default | Description |
| --- | --- | --- |
| `Store` | required | Where the token is kept. `NewCookieStore`, or your own `Store`. |
| `FieldName` | `csrf.token` | The form field the token is read from. |
| `RequestHeader` | `X-CSRF-Token` | The request header the token is read from. |
| `ErrorHandler` | `UnauthorizedHandler` | The handler of a refused request. |
| `TrustedOrigins` | none | Hosts, other than the request's own, that a `Referer` may come from. List only hosts you control. |

`New` returns a value with two handlers to choose from:

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/csrf"
)

func main() {
	app := iris.New()

	c := csrf.New(csrf.Options{
		Store: csrf.NewCookieStore(
			[]byte("9AB0F421E53A477C084477AEA06096F5"), csrf.Secure(false)),
		TrustedOrigins: []string{"ui.example.com"},
		ErrorHandler: func(ctx iris.Context) {
			ctx.Application().Logger().Warnf("csrf: %v", csrf.FailureReason(ctx))
			ctx.StopWithJSON(iris.StatusForbidden, iris.Map{"error": "forbidden"})
		},
	})

	// The middleware.
	app.Use(c.Protect)

	// Or the check alone, when the handler decides what a failure means.
	app.Post("/optional", func(ctx iris.Context) {
		if !c.Filter(ctx) {
			ctx.WriteString("not verified")
			return
		}
		ctx.WriteString("verified")
	})

	app.Listen(":8080")
}
```

### The default error response

`UnauthorizedHandler` answers `403 Forbidden` with the reason as plain text, one of:

| Error | Text |
| --- | --- |
| `ErrNoReferer` | `referer not supplied` |
| `ErrBadReferer` | `referer invalid` |
| `ErrNoToken` | `CSRF token not found in request` |
| `ErrBadToken` | `CSRF token invalid` |

When the token could not be generated or saved, the failure is the server's. The response is a 403 with no body, and `FailureReason(ctx)` returns the store's error for your logs.

### Cookie options

The cookie store issues a cookie named `_iris_csrf`, signed with the authentication key through [gorilla/securecookie](https://github.com/gorilla/securecookie).

| Option | Default | Description |
| --- | --- | --- |
| `CookieName(string)` | `_iris_csrf` | The cookie's name. |
| `Secure(bool)` | true | Send the cookie over HTTPS only. |
| `HTTPOnly(bool)` | true | Hide the cookie from scripts. |
| `SameSite(http.SameSite)` | `http.SameSiteLaxMode` | The SameSite attribute. `Strict` also keeps the cookie off requests that follow a link from another site. |
| `MaxAge(int)` | 43200 (12 hours) | Lifetime in seconds. 0 makes it a session cookie. |
| `Domain(string)` | the request's host | The cookie's domain. |
| `Path(string)` | the path the cookie was issued from | The cookie's path. Set `csrf.Path("/")` when forms and their targets live under different paths. |

## A custom store

A `Store` reads and saves the unmasked token for a request. Implement it to keep the token in a server-side session instead of a cookie:

```go
package main

import (
	"errors"

	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/csrf"
)

// memoryStore keeps the tokens in memory, by a session id that an
// earlier middleware is expected to have stored on the Context.
type memoryStore struct {
	tokens map[string][]byte
}

func (s *memoryStore) Get(ctx iris.Context) ([]byte, error) {
	token, ok := s.tokens[ctx.Values().GetString("session_id")]
	if !ok {
		return nil, errors.New("no token for this session")
	}
	return token, nil
}

func (s *memoryStore) Save(ctx iris.Context, token []byte) error {
	s.tokens[ctx.Values().GetString("session_id")] = token
	return nil
}

func main() {
	app := iris.New()

	store := &memoryStore{tokens: make(map[string][]byte)}
	app.Use(csrf.New(csrf.Options{Store: store}).Protect)

	app.Listen(":8080")
}
```

The map above is not safe for concurrent requests. A real store needs a lock or a database.

## Design notes

- The token is masked with a one-time pad on every request, which mitigates the [BREACH](http://breachattack.com/) attack. The unmasked token stays the same for the life of the cookie, so several open tabs keep working.
- Tokens and pads are 32 bytes from `crypto/rand`.
- Only the safe methods (GET, HEAD, OPTIONS, TRACE) skip the comparison. Everything else is checked.

## License

BSD licensed. See the [LICENSE](LICENSE) file.
