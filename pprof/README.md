## Middleware information

This folder contains a middleware which enables net/http/pprof.


## Install

```sh
$ go get -u gopkg.in/iris-contrib/middleware.v4/pprof
```

## Usage

```go
package main

import (
	"gopkg.in/iris-contrib/middleware.v4/pprof"
	"github.com/kataras/iris"
)

func main() {

	iris.Get("/", func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, "<h1> Please click <a href='/debug/pprof'>here</a>")
	})

	iris.Get("/debug/pprof/*action", pprof.New())

	iris.Listen(":8080")
}

```
