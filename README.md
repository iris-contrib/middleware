# Iris Community Middleware List

<!-- [![Build status](https://api.travis-ci.org/iris-contrib/middleware.svg?branch=master&style=flat-square)](https://travis-ci.org/iris-contrib/middleware) -->

This repository provides a way to share community-based middlewares for [Iris v12.2.0+ (currently `master` branch)](https://github.com/kataras/iris). Among with those, you can also navigate through the [builtin Iris handlers](https://github.com/kataras/iris/tree/v12/middleware).

## Installation

Install a middleware, take for example the [jwt](jwt) one.

```sh
$ go env -w GOPROXY=goproxy.cn,gocenter.io,goproxy.io,direct
$ go mod init myapp
$ go get github.com/kataras/iris/v12@master
$ go get github.com/iris-contrib/middleware/jwt@master
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

Middleware is just a chain handlers which can be executed before or after the main handler, can transfer data between handlers and communicate with third-party libraries, they are just functions.

<!-- https://github.com/kataras/iris/blob/master/_examples/permissions/main.go -->

| Middleware      | Description | Example     |
| ----------------|-------------|-------------|
| [pg](pg) | PostgreSQL Database | [pg/_examples](pg/_examples/) |
| [jwt](jwt) | JSON Web Tokens | [jwt/_example](jwt/_example/) |
| [cors](cors) | HTTP Access Control. | [cors/_example](cors/_example) |
| [secure](secure) | Middleware that implements a few quick security wins | [secure/_example](secure/_example/main.go) |
| [tollbooth](tollboothic) | Generic middleware to rate-limit HTTP requests | [tollboothic/_examples/limit-handler](tollboothic/_examples/limit-handler) |
| [cloudwatch](cloudwatch) |  AWS cloudwatch metrics middleware |[cloudwatch/_example](cloudwatch/_example) |
| [newrelic/v3](newrelic) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent/tree/master/v3) | [newrelic/_example](newrelic/_example) |
| [prometheus](prometheus)| Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool | [prometheus/_example](prometheus/_example) |
| [casbin](casbin)| An authorization library that supports access control models like ACL, RBAC, ABAC | [casbin/_examples](casbin/_examples) |
| [sentry-go (ex. raven)](https://github.com/getsentry/sentry-go/tree/master/iris)| Sentry client in Go | [sentry-go/example/iris](https://github.com/getsentry/sentry-go/blob/master/example/iris/main.go) | <!-- raven was deprecated by its company, the successor is sentry-go, they contain an Iris middleware. -->
| [csrf](csrf)| Cross-Site Request Forgery Protection | [csrf/_example](csrf/_example/main.go) |
| [throttler](throttler)| Rate limiting access to HTTP endpoints | [throttler/_example](throttler/_example/main.go) |
| [expmetric](expmetric)| Expvar for counting requests etc. | [expmetric/_example](expmetric/_example/main.go) |
| [zap](zap)| Provides log handling using zap package | [zap/_examples](zap/_examples/example_1/main.go) |

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

**To a party of routes or subdomain**

```go

p := app.Party("/sellers", authMiddleware, logMiddleware)

```

OR

```go
p := app.Party("/customers")
p.Use(logMiddleware)
```

**To all routes**

```go
app.Use(func(ctx iris.Context) { }, myMiddleware2)
```

**To global, all registered routes (including the http errors)**

```go
app.UseGlobal(func(ctx iris.Context) { }, myMiddleware2)
```

**To Party and its children, even on unmatched routes and errors**

```go
app.UseRouter(func(ctx iris.Context) { }, myMiddleware2))
```

## Can I use standard net/http handler with iris?

**Yes** you can, just pass the Handler inside the `iris.FromStd` in order to be converted into iris.Handler and register it as you saw before.

### Convert handler which has the form of `http.Handler/HandlerFunc`

```go
package main

import (
    "github.com/kataras/iris/v12"
)

func main() {
    app := iris.New()

    sillyHTTPHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
            println(r.RequestURI)
    })

    sillyConvertedToIon := iris.FromStd(sillyHTTPHandler)
    // FromStd can take (http.ResponseWriter, *http.Request, next http.Handler) too!
    app.Use(sillyConvertedToIon)

    app.Listen(":8080")
}

```

## Contributing

If you are interested in contributing to this project, please push a PR.

## People

[List of all contributors](https://github.com/iris-contrib/middleware/graphs/contributors)
