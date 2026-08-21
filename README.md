# Iris Community Middleware List

This repository holds community middlewares for the [Iris Web Framework](https://github.com/kataras/iris). The framework also ships its own: see the [builtin Iris middleware](https://github.com/kataras/iris/tree/main/middleware) before reaching for a community one.

## Iris version

These modules target **Iris v14** (`github.com/kataras/iris/v14`) on the `main`
branch. The v12 line is preserved unchanged at the [`v12`](https://github.com/iris-contrib/middleware/tree/v12)
tag; use that if your application has not migrated yet. The
[migration guide](https://github.com/kataras/iris/blob/main/MIGRATION.md) covers the upgrade.

Two modules here, `cors` and `jwt`, predate the framework's own. On v14 prefer
the built-in [`middleware/cors`](https://github.com/kataras/iris/tree/main/middleware/cors)
and [`middleware/jwt`](https://github.com/kataras/iris/tree/main/middleware/jwt),
which are maintained with the framework.

## Installation

Install a middleware, for example the [jwt](jwt) one:

```sh
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest
$ go get github.com/iris-contrib/middleware/jwt@main
```

**import as**

```go
import "github.com/iris-contrib/middleware/jwt"

// [...Code]
```

**build**

```sh
$ go build
```

A middleware is a plain `iris.Handler`. Handlers form a chain around the main route handler: each one can run code before or after the rest of the chain, share data through the context and talk to third-party libraries.

| Middleware      | Description | Example     |
| ----------------|-------------|-------------|
| [pg](pg) | PostgreSQL database access with a generic repository | [pg/_examples](pg/_examples/) |
| [jwt](jwt) | JSON Web Tokens | [jwt/_example](jwt/_example/) |
| [cors](cors) | HTTP access control | [cors/_example](cors/_example) |
| [secure](secure) | A few quick security wins: security response headers | [secure/_example](secure/_example/main.go) |
| [tollbooth](tollboothic) | Generic middleware to rate-limit HTTP requests | [tollboothic/_examples/limit-handler](tollboothic/_examples/limit-handler) |
| [cloudwatch](cloudwatch) | AWS CloudWatch metrics | [cloudwatch/_example](cloudwatch/_example) |
| [newrelic/v3](newrelic) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent/tree/master/v3) | [newrelic/_example](newrelic/_example) |
| [prometheus](prometheus) | Metrics endpoint for the [Prometheus](http://prometheus.io) instrumentation tool | [prometheus/_example](prometheus/_example) |
| [casbin](casbin) | Authorization with access control models: ACL, RBAC, ABAC | [casbin/_examples](casbin/_examples) |
| [sentry-go (ex. raven)](https://github.com/getsentry/sentry-go/tree/master/iris) | Sentry client in Go | [sentry-go/example/iris](https://github.com/getsentry/sentry-go/blob/master/example/iris/main.go) |
| [csrf](csrf) | Cross-site request forgery protection | [csrf/_example](csrf/_example/main.go) |
| [throttler](throttler) | Rate limiting access to HTTP endpoints | [throttler/_example](throttler/_example/main.go) |
| [expmetric](expmetric) | Expvar metrics, for counting requests etc. | [expmetric/_example](expmetric/_example/main.go) |
| [zap](zap) | Request logging and panic recovery through [zap](https://github.com/uber-go/zap) | [zap/_examples](zap/_examples/example_1/main.go) |

### Register a middleware

**To a single route**

```go
app := iris.New()
app.Get("/mypath",
  onBegin,
  mySecondMiddleware,
  mainHandler,
)

func onBegin(ctx iris.Context) { /* ... */ ctx.Next() }
func mySecondMiddleware(ctx iris.Context) { /* ... */ ctx.Next() }
func mainHandler(ctx iris.Context) { /* ... */ }
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

**To all routes of the application, including the not-found and method-not-allowed handlers**

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

`UseRouter` handlers run even before routing, so they fire for every request that reaches the server. That is the place for CORS, panic recovery and access logs.

## Can I use a standard net/http handler with Iris?

**Yes.** Pass the handler to `iris.FromStd` to convert it into an `iris.Handler` and register it as shown above.

### Convert a handler of the form `http.Handler/HandlerFunc`

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
    app.Use(iris.FromStd(stdHandler))

    app.Listen(":8080")
}
```

## Contributing

If you are interested in contributing to this project, please push a PR.

## People

[List of all contributors](https://github.com/iris-contrib/middleware/graphs/contributors)
