# newrelic

[New Relic](https://newrelic.com) instrumentation for [Iris](https://github.com/kataras/iris) v14, through the official [Go agent](https://github.com/newrelic/go-agent) v3.

The middleware starts a web transaction for each request, reports the response status code to it and stores it on the Context for the handlers.

## Install

```sh
go get github.com/iris-contrib/middleware/newrelic@main
```

```go
import nriris "github.com/iris-contrib/middleware/newrelic"
```

The package is named `newrelic`, like the agent's own. Import it under another name.

## Example

```go
package main

import (
	"fmt"
	"os"

	"github.com/kataras/iris/v14"

	nriris "github.com/iris-contrib/middleware/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Iris App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	app := iris.New()
	// On the root application Use also runs for requests that match no
	// route, so 404 answers are instrumented too.
	app.Use(nriris.MiddlewareWithFullPath(nrApp))

	// Reported as "GET /users/{id:int}", whatever the id.
	app.Get("/users/{id:int}", func(ctx iris.Context) {
		ctx.Writef("user %d", ctx.Params().GetInt("id"))
	})

	app.Get("/report", func(ctx iris.Context) {
		// The transaction of the current request.
		txn := nriris.Transaction(ctx)
		txn.AddAttribute("report", "monthly")

		ctx.WriteString("done")
	})

	app.Listen(":8000")
}
```

## Functions

| Name | Description |
| --- | --- |
| `Middleware(app *newrelic.Application) iris.Handler` | Names the transaction after the request method and the name of the next handler in the chain. |
| `New(app)` | The same as `Middleware`. |
| `MiddlewareWithFullPath(app)` | Names the transaction after the request method and the path of the route as it was registered, for example `GET /users/{id:int}`. A request that matched no route is named after the method and the full request URI. |
| `Transaction(ctx) *newrelic.Transaction` | The transaction of the current request, or nil when the middleware did not run or its `app` was nil. |
| `GetTransaction(ctx)` | The same as `Transaction`, for an `iris.Context`. |
| `TransactionContextKey` | The Context key the transaction is stored under. |

`MiddleareWithFullPath`, the earlier misspelled name, still works and is deprecated.

A nil `*newrelic.Application` turns the middleware into a handler that only calls the next one. That is how to switch instrumentation off in tests or in development.

## Transaction names

`Middleware` takes the name from the handler that follows it in the chain. When more middleware is registered after it, that is the name of the next middleware and not of the route's handler, and every route ends up under one name. Register it last among your middleware, or use `MiddlewareWithFullPath`.

An anonymous handler function has a compiler-generated name such as `main.main.func2`. Named functions give readable transaction names.

## Tests

The tests use the agent's `integrationsupport` test application, which never connects and keeps what it would have sent in memory. Transaction names, the reported error for a 5xx response and the wiring of the response writer are covered. Sending the data to New Relic is not.
