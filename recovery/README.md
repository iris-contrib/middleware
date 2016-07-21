## Middleware information

This folder contains a middleware for safety recover the server from panic

## Install

```sh
$ go get -u github.com/iris-contrib/middleware/recovery
```

## How to use

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/iris-contrib/middleware/recovery"
)

func main() {

	iris.Use(recovery.New(iris.Logger)) // optional parameter is the logger which the stack of the panic will be printed

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("Hi, let's panic")
		panic("errorrrrrrrrrrrrrrr")
	})

	iris.Listen(":8080")
}

```
