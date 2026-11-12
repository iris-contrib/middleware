# prometheus

Request metrics for [Iris](https://github.com/kataras/iris) v14 in the [Prometheus](https://prometheus.io) format, through [client_golang](https://github.com/prometheus/client_golang).

The middleware records two metrics, each with the labels `code`, `method`, `path` and `service`:

| Metric | Type | Description |
| --- | --- | --- |
| `http_requests_total` | counter | Number of requests. |
| `http_request_duration_seconds` | histogram | Time the handlers took. |

## Install

```sh
go get github.com/iris-contrib/middleware/prometheus@main
```

```go
import prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"
```

The package is named `prometheus`, like the client library's own. Import one of the two under another name.

## Example

```go
package main

import (
	"github.com/kataras/iris/v14"

	prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	app := iris.New()

	// The service name, then the histogram buckets in seconds.
	m := prometheusMiddleware.New("my_service", 0.3, 1.2, 5.0)

	// On the root application Use records every request,
	// the ones that match no route included.
	app.Use(m.ServeHTTP)

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello")
	})

	// The page that the Prometheus server scrapes.
	app.Get("/metrics", iris.FromStd(promhttp.Handler()))

	app.Listen(":8080")
}
```

After two requests to `/`, the `/metrics` page contains:

```text
http_requests_total{code="200",method="GET",path="/",service="my_service"} 2
```

## API

| Name | Description |
| --- | --- |
| `New(name string, buckets ...float64) *Prometheus` | Creates the two collectors and registers them to the default Prometheus registry. `name` is the value of the `service` label. Without buckets, `DefaultBuckets` is used. |
| `(*Prometheus).ServeHTTP` | The handler. Runs the rest of the chain, then records the request with its final status code. |
| `DefaultBuckets` | `0.3, 1.2, 5.0` seconds. |

## Things to know before you use it

- Call `New` once per service name. A second call with the same name panics, because the collectors are already registered.
- On Iris v12 the not found handler had to call `m.ServeHTTP` itself. On v14 `app.Use` on the root application already covers it. If your `HandleNoRoute` handler still calls `m.ServeHTTP`, each 404 is counted twice.
- The `path` label is the request path, not the route. `/users/1` and `/users/2` are two series, and a client that requests random paths creates a new series for each one. On a public server, register with `app.UseRelevant` or on the routers you want measured, so that only matched routes are recorded.
