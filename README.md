This repository provides a way to share any minor handlers for [iris](https://github.com/kataras/iris) web framework. You can view the built'n supported handlers by pressing [here](https://github.com/kataras/iris/tree/master/middleware).

[![Build status](https://api.travis-ci.org/iris-contrib/middleware.svg?branch=master&style=flat-square)](https://travis-ci.org/iris-contrib/middleware)

## Installation

```sh
$ go get github.com/iris-contrib/middleware/...
```

Middleware is just a chain handlers which can be executed before or after the main handler, can transfer data between handlers and communicate with third-party libraries, they are just functions.

| Middleware | Description | Example |
| -----------|--------|-------------|
| [jwt](https://github.com/iris-contrib/middleware/tree/master/jwt) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it. | [jwt/_example](https://github.com/iris-contrib/middleware/tree/master/jwt/_example) |
| [cors](https://github.com/iris-contrib/middleware/tree/master/cors) | HTTP Access Control. | [cors/_example](https://github.com/iris-contrib/middleware/tree/master/cors/_example) |
| [secure](https://github.com/iris-contrib/middleware/tree/master/secure) | Middleware that implements a few quick security wins. | [secure/_example](https://github.com/iris-contrib/middleware/tree/master/secure/_example/main.go) |
| [tollbooth](https://github.com/iris-contrib/middleware/tree/master/tollboothic) | Generic middleware to rate-limit HTTP requests. | [tollbooth/_examples/limit-handler](https://github.com/iris-contrib/middleware/tree/master/tollbooth/_examples/limit-handler) |
| [cloudwatch](https://github.com/iris-contrib/middleware/tree/master/cloudwatch) |  AWS cloudwatch metrics middleware. |[cloudwatch/_example](https://github.com/iris-contrib/middleware/tree/master/cloudwatch/_example) |
| [new relic](https://github.com/iris-contrib/middleware/tree/master/newrelic) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent). | [newrelic/_example](https://github.com/iris-contrib/middleware/tree/master/newrelic/_example) |
| [prometheus](https://github.com/iris-contrib/middleware/tree/master/prometheus)| Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool | [prometheus/_example](https://github.com/iris-contrib/middleware/tree/master/prometheus/_example) |

### How can I register middleware?

**To a single route**
```go
app := iris.New()
app.Get("/mypath", myMiddleware1, myMiddleware2, func(ctx context.Context){}, func(ctx context.Context){}, myMiddleware5,myMainHandlerLast)
```

**To a party of routes or subdomain**
```go

myparty := app.Party("/myparty", myMiddleware1,func(ctx context.Context){},myMiddleware3)
{
	//....
}

```

**To all routes**
```go
app.Use(func(ctx context.Context){}, myMiddleware2)
```

**To global, all routes on all subdomains on all parties**
```go
app.UseGlobal(func(ctx context.Context){}, myMiddleware2)
```


## Can I use standard net/http handler with iris?

**Yes** you can, just pass the Handler inside the `iris.FromStd` in order to be converted into context.Handler and register it as you saw before.

### Convert handler which has the form of `http.Handler/HandlerFunc`

```go
package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	sillyHTTPHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	     println(r.RequestURI)
	})

	sillyConvertedToIon := iris.FromStd(sillyHTTPHandler)
	// FromStd can take (http.ResponseWriter, *http.Request, next http.Handler) too!
	app.Use(sillyConvertedToIon)

	app.Run(iris.Addr(":8080"))
}

```

## Contributing

If you are interested in contributing to this project, please push a PR.

## People

[List of all contributors](https://github.com/iris-contrib/middleware/graphs/contributors)