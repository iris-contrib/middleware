# expmetric

Request counters for [Iris](https://github.com/kataras/iris) v14, published through the standard library's [expvar](https://pkg.go.dev/expvar) package.

Each function below registers an expvar variable and returns the handler that updates it. `expmetric.Handler()` serves all of them as JSON.

## Install

```sh
go get github.com/iris-contrib/middleware/expmetric@main
```

```go
import "github.com/iris-contrib/middleware/expmetric"
```

## Example

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/expmetric"
)

func main() {
	app := iris.New()

	// Lists every expvar as JSON. Protect it in production.
	app.Get("/debug/vars", expmetric.Handler())

	// On the root application Use counts every request,
	// the ones that match no route included.
	app.Use(expmetric.HitsTotal())

	// A counter for one route, under a name of your choice.
	app.Get("/", expmetric.HitsPerMinute(expmetric.MetricName("home_hits_per_minute")), func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	// Open http://localhost:8080 and then http://localhost:8080/debug/vars
	app.Listen(":8080")
}
```

To count matched routes only, register with `app.UseRelevant` or on a child router.

## Counters

| Function | Variable | Default name |
| --- | --- | --- |
| `HitsTotal(options...)` | Number of requests since the start. | `hits_total` |
| `HitsPerSecond(options...)` | Requests in the last second. | `hits_per_second` |
| `HitsPerMinute(options...)` | Requests in the last minute. Also registers `<name>_avg`, the value divided by 60. | `hits_per_minute` |
| `HitsPerHour(options...)` | Requests in the last hour. Also registers `<name>_avg`, the value divided by 24. | `hits_per_hour` |
| `HitsTotalPerPath(options...)` | A map of request path to number of requests. | none, `MetricName` is required |
| `HitsTotalPerUserID(options...)` | A map of user id to number of requests. | none, `MetricName` is required |
| `HitsTotalPerUserEmail(options...)` | A map of user email to number of requests. | none, `MetricName` is required |
| `HitsTotalWithKeyFunc(keyFunc, options...)` | A map of your own key to number of requests. An empty key is not counted. | none, `MetricName` is required |
| `KeyValue(keyFunc, valueFunc, options...)` | A map of your own key to your own value. | none, `MetricName` is required |

The two per-user counters read `ctx.User()`, which an authentication middleware sets. A request without a user is served and not counted.

A `ValueFunc` returns the value and whether to store it. A string, an `int64`, an `expvar.Var`, a `json.Marshaler` and any value that encodes to JSON are accepted.

## Options

| Option | Description |
| --- | --- |
| `MetricName(name string)` | The name of the expvar variable. |
| `Resolution(n int)` | How many slices the time window of a rate counter is divided into. Defaults to 20. |
| `Avg(n int64)` | For `HitsTotal` and `HitsPerSecond`: registers a second variable, `<name>_avg` (or `<name>.avg` when the name contains a dot), with the value divided by `n`. `HitsPerMinute` and `HitsPerHour` always use 60 and 24. The map counters ignore it. |

## Limits

- expvar names are global to the process. Registering the same name twice panics, so create each counter once, at startup.
- `HitsTotalPerPath` keeps one entry per distinct request path. Registered with `app.Use` on a public server it grows with every path that a client invents. Use `app.UseRelevant`, or a key function that returns the route instead of the path.
