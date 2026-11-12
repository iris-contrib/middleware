# Iris Community Middleware

Community middleware for the [Iris Web Framework](https://github.com/kataras/iris). Thirteen Go modules, one per directory, each installed on its own.

The framework ships [middleware of its own](https://github.com/kataras/iris/tree/main/middleware). Check there first: three things people come here for are built in, see [below](#already-in-the-framework).

## Iris version

The `main` branch targets **Iris v14** (`github.com/kataras/iris/v14`) and needs Go 1.27 or later. The framework's [migration guide](https://github.com/kataras/iris/blob/main/MIGRATION.md) covers the upgrade from v12.

The last revision for Iris v12 is kept on the [`v12` branch](https://github.com/iris-contrib/middleware/tree/v12). Go cannot select a branch whose name looks like a version (`go get ...@v12` fails with "no matching versions"), so install that revision by the commit the branch points at:

```sh
go get github.com/iris-contrib/middleware/jwt@92b8edf402f2
```

## Modules

| Module | Description | Example |
| --- | --- | --- |
| [casbin](https://github.com/iris-contrib/middleware/tree/main/casbin) | Authorization with [Casbin](https://github.com/casbin/casbin) models: ACL, RBAC, ABAC | [casbin/_examples](https://github.com/iris-contrib/middleware/tree/main/casbin/_examples) |
| [cloudwatch](https://github.com/iris-contrib/middleware/tree/main/cloudwatch) | Request latency metrics sent to AWS CloudWatch | [cloudwatch/_example](https://github.com/iris-contrib/middleware/tree/main/cloudwatch/_example) |
| [cors](https://github.com/iris-contrib/middleware/tree/main/cors) | CORS headers and preflight answers, a port of rs/cors | [cors/_example](https://github.com/iris-contrib/middleware/tree/main/cors/_example) |
| [csrf](https://github.com/iris-contrib/middleware/tree/main/csrf) | Cross-site request forgery protection with a signed cookie token | [csrf/_example](https://github.com/iris-contrib/middleware/tree/main/csrf/_example) |
| [expmetric](https://github.com/iris-contrib/middleware/tree/main/expmetric) | Request counters published through `expvar` | [expmetric/_example](https://github.com/iris-contrib/middleware/tree/main/expmetric/_example) |
| [jwt](https://github.com/iris-contrib/middleware/tree/main/jwt) | JSON Web Token verification on golang-jwt | [jwt/_example](https://github.com/iris-contrib/middleware/tree/main/jwt/_example) |
| [newrelic](https://github.com/iris-contrib/middleware/tree/main/newrelic) | New Relic transactions through the official [Go agent](https://github.com/newrelic/go-agent) v3 | [newrelic/_example](https://github.com/iris-contrib/middleware/tree/main/newrelic/_example) |
| [pg](https://github.com/iris-contrib/middleware/tree/main/pg) | PostgreSQL through [kataras/pg](https://github.com/kataras/pg): per-request transactions, typed repositories, a CRUD controller | [pg/_examples](https://github.com/iris-contrib/middleware/tree/main/pg/_examples) |
| [prometheus](https://github.com/iris-contrib/middleware/tree/main/prometheus) | Request counter and latency histogram for [Prometheus](https://prometheus.io) | [prometheus/_example](https://github.com/iris-contrib/middleware/tree/main/prometheus/_example) |
| [secure](https://github.com/iris-contrib/middleware/tree/main/secure) | Security response headers, allowed hosts and HTTPS redirect, a port of unrolled/secure | [secure/_example](https://github.com/iris-contrib/middleware/tree/main/secure/_example) |
| [throttler](https://github.com/iris-contrib/middleware/tree/main/throttler) | Rate limiting through [throttled](https://github.com/throttled/throttled), with its pluggable stores | [throttler/_example](https://github.com/iris-contrib/middleware/tree/main/throttler/_example) |
| [tollboothic](https://github.com/iris-contrib/middleware/tree/main/tollboothic) | Rate limiting through [tollbooth](https://github.com/didip/tollbooth) | [tollboothic/_examples](https://github.com/iris-contrib/middleware/tree/main/tollboothic/_examples) |
| [zap](https://github.com/iris-contrib/middleware/tree/main/zap) | Request logging and panic recovery through [zap](https://github.com/uber-go/zap) | [zap/_examples](https://github.com/iris-contrib/middleware/tree/main/zap/_examples) |

Each module has a README with its install command, an example and its options.

Maintained elsewhere: [sentry-go/iris](https://github.com/getsentry/sentry-go/tree/master/iris), the Sentry client's own Iris middleware.

### Already in the framework

| You want | Use |
| --- | --- |
| CORS | [`middleware/cors`](https://github.com/kataras/iris/tree/main/middleware/cors). The `cors` module here predates it and is kept for applications that already use it. |
| JWT | [`middleware/jwt`](https://github.com/kataras/iris/tree/main/middleware/jwt). It signs, verifies, reads claims into your own types and has a blocklist. The `jwt` module here only verifies, and is kept for applications that already use it. |
| An API key check | [`middleware/apikey`](https://github.com/kataras/iris/tree/main/middleware/apikey). This repository has never had a module for it. |

The framework also has [`middleware/rate`](https://github.com/kataras/iris/tree/main/middleware/rate) for rate limiting in memory, and [`middleware/accesslog`](https://github.com/kataras/iris/tree/main/middleware/accesslog) with [`middleware/recovery`](https://github.com/kataras/iris/tree/main/middleware/recovery) for request logs and panics. `throttler`, `tollboothic` and `zap` cover the same ground through other libraries; their READMEs say when to pick which.

## Installation

Every directory is a separate Go module. Install the ones you use, here [jwt](https://github.com/iris-contrib/middleware/tree/main/jwt):

```sh
go mod init myapp
go get github.com/kataras/iris/v14@latest
go get github.com/iris-contrib/middleware/jwt@main
```

The modules have no version tags. `@main` resolves to the latest commit of the `main` branch and `go.mod` records it as a pseudo-version, so your build stays on that commit until you run `go get` again.

Import it:

```go
import "github.com/iris-contrib/middleware/jwt"
```

## Register a middleware

A middleware is a plain `iris.Handler`. Handlers form a chain around the main route handler: each one can run code before or after the rest of the chain, share data through the Context and stop the request.

**To a single route**

```go
package main

import "github.com/kataras/iris/v14"

func main() {
	app := iris.New()
	app.Get("/mypath",
		onBegin,
		mySecondMiddleware,
		mainHandler,
	)

	app.Listen(":8080")
}

func onBegin(ctx iris.Context)            { /* ... */ ctx.Next() }
func mySecondMiddleware(ctx iris.Context) { /* ... */ ctx.Next() }
func mainHandler(ctx iris.Context)        { /* ... */ }
```

**To a group of routes or a subdomain**

```go
sellers := app.Router("/sellers", authMiddleware, logMiddleware)
```

OR

```go
customers := app.Router("/customers")
customers.Use(logMiddleware)
```

**To all requests of the application, the ones that match no route (404, 405) included**

```go
app.Use(myMiddleware)
```

**To matched routes only**

```go
app.UseRelevant(myMiddleware)
```

**Before the router itself, on every request**

```go
app.UseRouter(myMiddleware)
```

`UseRouter` handlers run before routing, so they can change the request path and they see every request that reaches the server. That is the place for CORS, panic recovery and access logs.

On Iris v12 `app.Use` ran on matched routes only. On v14 it also runs for 404 and 405 on the root application, and keeps its v12 meaning on a child router. If a middleware of this repository behaves differently after your upgrade, look there first: `app.UseRelevant` is the v12 behavior.

## Can I use a standard net/http handler with Iris?

**Yes.** Pass the handler to `iris.FromStd` to convert it into an `iris.Handler` and register it as shown above.

```go
package main

import (
	"net/http"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	stdHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		println(r.RequestURI)
	})

	// FromStd also accepts the
	// func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) form.
	app.Get("/std", iris.FromStd(stdHandler))

	app.Listen(":8080")
}
```

## Development

The modules are tested on every push and pull request to `main` by [.github/workflows/ci.yml](https://github.com/iris-contrib/middleware/blob/main/.github/workflows/ci.yml): `go build`, `go vet` and `go test -race` for each module, with the Go version of its `go.mod`, plus a build of its examples. No test needs a network connection, a database or cloud credentials.

To run the same for one module:

```sh
cd csrf
go build ./... && go vet ./... && go test -race ./...
```

## Contributing

Pull requests are welcome. A new middleware goes in a directory of its own with a `go.mod`, a README, an example and tests.

## People

[List of all contributors](https://github.com/iris-contrib/middleware/graphs/contributors)
