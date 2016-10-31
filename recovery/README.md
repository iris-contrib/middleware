## Middleware information

This folder contains a middleware for safety recover the server from panic

## Install

```sh
$ go get -u gopkg.in/iris-contrib/middleware.v4/recovery
```

## How to use

```go

package main

import (
	"gopkg.in/kataras/iris.v4"
	"gopkg.in/iris-contrib/middleware.v4/recovery"
)

func main() {

	iris.Use(recovery.Handler)
	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("Hi, let's panic")
		panic("errorrrrrrrrrrrrrrr")
	})

	iris.Listen(":8080")
}

```
