# cors

Cross-Origin Resource Sharing for [Iris](https://github.com/kataras/iris) v14. A port of [rs/cors](https://github.com/rs/cors).

**Iris has its own CORS middleware**, [kataras/iris/middleware/cors](https://github.com/kataras/iris/tree/main/middleware/cors), which is maintained with the framework. Prefer it for new code. This module remains for applications that already use its `Options`.

## Install

```sh
go get github.com/iris-contrib/middleware/cors@main
```

```go
import "github.com/iris-contrib/middleware/cors"
```

## Example

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/cors"
)

func main() {
	app := iris.New()

	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           600,
	})

	// UseRouter runs the handler before the router, so it also answers the
	// preflight (OPTIONS) request of a path that has no OPTIONS route.
	app.UseRouter(crs)

	app.Post("/api/send", func(ctx iris.Context) {
		ctx.WriteString("sent")
	})

	app.Listen(":8080")
}
```

To register it for a group of routes instead, let the group answer OPTIONS as well:

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/cors"
)

func main() {
	app := iris.New()

	api := app.Router("/api")
	// Every route registered below is also registered for OPTIONS,
	// which is what the browser's preflight request needs.
	api.AllowMethods(iris.MethodOptions)
	api.Use(cors.Default())

	api.Get("/items", func(ctx iris.Context) {
		ctx.WriteString("items")
	})

	app.Listen(":8080")
}
```

## Constructors

| Function | Description |
| --- | --- |
| `New(Options) iris.Handler` | The handler for the given options. |
| `Default() iris.Handler` | `New(Options{})`: any origin, the methods GET, POST and HEAD, no credentials. |
| `AllowAll() iris.Handler` | Any origin, six methods, any header, with credentials. See the warning below. |

## Options

| Field | Default | Description |
| --- | --- | --- |
| `AllowedOrigins []string` | any origin | Origins that may make cross-origin requests. `*` allows all. An origin may contain one `*` wildcard, as in `https://*.example.com`. |
| `AllowOriginFunc func(origin string) bool` | none | Decides per origin. When set, `AllowedOrigins` is ignored. |
| `AllowedMethods []string` | `GET`, `POST`, `HEAD` | Methods a cross-origin request may use. |
| `AllowedHeaders []string` | `Origin`, `Accept`, `Content-Type`, `X-Requested-With` | Request headers a cross-origin request may send. `*` allows all. `Origin` is always added to your list. |
| `ExposedHeaders []string` | none | Response headers that scripts may read. |
| `MaxAge int` | 0, header not sent | Seconds a browser may cache a preflight answer. |
| `AllowCredentials bool` | false | Sends `Access-Control-Allow-Credentials: true`. |
| `OptionsPassthrough bool` | false | After the preflight headers are set, calls the next handler instead of answering 200. |
| `Debug bool` | false | Logs each decision to the standard output. |

## Behavior

- A request from an origin, or with a method, that the options do not allow is answered with `403 Forbidden` and the route's handler does not run. Browsers send the `Origin` header on same-origin POST requests too, so list your own origin when you list any.
- `Vary: Origin` is added to every answer, and the three preflight header names to a preflight answer. The values are appended, so a `Vary` set by another middleware survives.
- A preflight request is answered with `200 OK` and no body.

### Credentials and the wildcard origin

Browsers refuse `Access-Control-Allow-Origin: *` together with `Access-Control-Allow-Credentials: true`. When `AllowCredentials` is true and every origin is allowed (`*`, or an empty `AllowedOrigins`), this middleware answers with the origin of the request instead of `*`. The browser accepts that, which means any website can then make requests with the visitor's cookies and read the answers. `AllowAll()` is configured exactly like that.

Do not combine the two for an API that authenticates with cookies or HTTP authentication. List the origins you trust. The framework's own middleware has `cors.Strict()` for this case.

## Examples

- [_example](_example/main.go): the server
- [_example/client](_example/client/main.go): a page on another origin that calls it
- [_example/manually](_example/manually/main.go): the headers set by hand, without the middleware
