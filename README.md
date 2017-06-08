<a href="https://travis-ci.org/iris-contrib/middleware"><img src="https://img.shields.io/travis/iris-contrib/middleware.svg?style=flat-square" alt="Build Status"></a>
<a href="https://github.com/kataras/iris/blob/master/HISTORY.md"><img src="https://img.shields.io/badge/release-v7%20-blue.svg?style=flat-square" alt="CHANGELOG/HISTORY"></a>


This repository provides a way to share your [Iris](https://github.com/kataras/iris)' specific middleware with the rest of us. You can view the @kataras supported middleware by pressing [here](https://github.com/kataras/iris/tree/master/middleware).


Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least version 1.8

```bash
$ go get github.com/iris-contrib/middleware/...
```


FAQ
------------
Explore [these questions](https://github.com/iris-contrib/middleware/issues) or navigate to the [community chat][Chat].


People
------------
The Community.



[Travis Widget]: https://img.shields.io/travis/iris-contrib/middleware.svg?style=flat-square
[Travis]: http://travis-ci.org/iris-contrib/middleware
[Release Widget]: https://img.shields.io/badge/release-v7-blue.svg?style=flat-square
[Release]: https://github.com/iris-contrib/adaptors/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/iris


# What?

Middleware are just handlers which can be served before or after the main handler, can transfer data between handlers and communicate with third-party libraries, they are just functions.

### How can I install a middleware?

```sh
$ go get -u github.com/iris-contrib/middleware/$FOLDERNAME
```

**NOTE**: When you install one middleware you will have all of them downloaded & installed, **no need** to re-run the go get foreach middeware.

### How can I register middleware?


**To a single route**
```go
app := iris.New()
app.Get("/mypath",myMiddleware1,myMiddleware2,func(ctx context.Context){}, func(ctx context.Context){},myMiddleware5,myMainHandlerLast)
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


# Can I use standard net/http handler with Iris?

**Yes** you can, just pass the Handler inside the `handlerconv.FromStd` in order to be converted into iris.HandlerFunc and register it as you saw before.

## handler which has the form of http.Handler/HandlerFunc

```go
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/handlerconv"
)

func main() {
	app := iris.New()

	sillyHTTPHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	     println(r.RequestURI)
	})

	sillyConvertedToIris := handlerconv.FromStd(sillyHTTPHandler)
	// FromStd can take (http.ResponseWriter, *http.Request, next http.Handler) too!
	app.Use(sillyConvertedToIris)

	app.Run(iris.Addr(":8080"))
}

```
