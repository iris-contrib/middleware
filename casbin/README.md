# casbin

Authorization for [Iris](https://github.com/kataras/iris) v14 through [Casbin](https://github.com/casbin/casbin) v2.

For each request the middleware asks a Casbin enforcer whether the subject may access the request path with the request method. If not, it answers `403 Forbidden`. The model and the policy decide everything else, so read the [Casbin documentation](https://casbin.org/docs/overview) first.

## Install

```sh
go get github.com/iris-contrib/middleware/casbin@main
```

```go
import "github.com/iris-contrib/middleware/casbin"
```

## Example

The model and policy files are the ones in [_examples/router](_examples/router).

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/basicauth"

	"github.com/iris-contrib/middleware/casbin"
)

func main() {
	app := iris.New()

	casbinMiddleware, err := casbin.NewEnforcer("casbinmodel.conf", "casbinpolicy.csv")
	if err != nil {
		panic(err)
	}

	// Authentication runs first. basicauth sets the request's user,
	// and the username is the subject that Casbin is asked about.
	auth := basicauth.Default(map[string]string{
		"alice": "alicepass",
		"bob":   "bobpass",
	})

	app.UseRouter(auth, casbinMiddleware.ServeHTTP)

	app.Get("/dataset1/{p:path}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", casbin.Subject(ctx))
	})

	app.Listen(":8080")
}
```

## Where to register it

An authentication middleware that sets the subject must run before `ServeHTTP`. Where the two are registered decides what a path with no route answers.

| Registration | Runs on | A path with no route answers |
| --- | --- | --- |
| `app.UseRouter(auth, c.ServeHTTP)` | every request, before the router | 403, unless a policy allows it |
| `app.Use(auth, c.ServeHTTP)` on the root application | every request, 404 and 405 included | 403, unless a policy allows it |
| `app.UseRelevant(auth, c.ServeHTTP)`, `Use` on a child router, or a single route | matched routes only | 404 |

On Iris v12 `app.Use` ran on matched routes only. If you relied on that, switch to `app.UseRelevant`.

## The subject

`Casbin.SubjectExtractor` returns the subject of a request. The default one calls `casbin.Subject(ctx)`, which returns:

1. the value a previous handler set with `casbin.SetSubject(ctx, subject)`, or
2. the username of `ctx.User()`, which is what `basicauth`, `jwt` and other authentication middleware set.

A request with no subject is checked with an empty one, which no policy normally allows.

## API

| Name | Description |
| --- | --- |
| `New(e *casbin.Enforcer) *Casbin` | Wraps an enforcer you built yourself, for example one with a database adapter. |
| `NewEnforcer(modelFile, policyFile string) (*Casbin, error)` | Builds the enforcer from a model file and a policy file. |
| `(*Casbin).ServeHTTP` | The handler. Calls `UnauthorizedHandler` when access is denied. |
| `(*Casbin).Check(ctx) bool` | The decision alone, for use inside a handler or with `iris.NewConditionalHandler`. |
| `(*Casbin).Enforce(ctx, subject) (bool, error)` | Asks the enforcer about a subject of your choice. |
| `Casbin.SubjectExtractor` | `func(iris.Context) string`. See above. |
| `Casbin.UnauthorizedHandler` | `iris.Handler` that answers a denied request. Defaults to a 403 with an empty body. |
| `SetSubject(ctx, subject)`, `Subject(ctx)` | Set and read the subject of the current request. |

When the enforcer itself fails (a matcher that does not evaluate, an adapter that cannot be reached) the request is denied. The error is stored on the Context as a private error: `ctx.GetErr()` returns it for your logs, and it is not sent to the client.

## Examples

- [_examples/router](_examples/router/main.go): registered with `UseRouter`
- [_examples/middleware](_examples/middleware/main.go): registered with `UseRelevant`

Each example has its own model, policy and test.
