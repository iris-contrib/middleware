# zap

Request logging and panic recovery for [Iris](https://github.com/kataras/iris) v14 through [zap](https://github.com/uber-go/zap). A port of [gin-contrib/zap](https://github.com/gin-contrib/zap), after the feature request [kataras/iris#2125](https://github.com/kataras/iris/issues/2125).

Iris has an access log and a recovery middleware of its own, [middleware/accesslog](https://github.com/kataras/iris/tree/main/middleware/accesslog) and [middleware/recovery](https://github.com/kataras/iris/tree/main/middleware/recovery). Use this module when the rest of your application already logs through zap.

## Install

```sh
go get github.com/iris-contrib/middleware/zap@main
```

The package is named `iriszap`:

```go
import iriszap "github.com/iris-contrib/middleware/zap"
```

## Example

```go
package main

import (
	"fmt"
	"time"

	iriszap "github.com/iris-contrib/middleware/zap"
	"github.com/kataras/iris/v14"
	"go.uber.org/zap"
)

func main() {
	app := iris.New()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Logs each request after it has been served:
	// RFC 3339 timestamps, in UTC.
	app.Use(iriszap.New(logger, time.RFC3339, true))

	// Recovers from a panic, logs it with the stack trace
	// and answers 500.
	app.Use(iriszap.RecoveryWithZap(logger, true))

	app.Get("/ping", func(ctx iris.Context) {
		ctx.Text("pong " + fmt.Sprint(time.Now().Unix()))
	})

	app.Get("/panic", func(ctx iris.Context) {
		panic("An unexpected error happen!")
	})

	app.Listen(":8080")
}
```

On the root application `Use` also runs for requests that match no route, so 404 answers are logged as well.

## Request logging

| Function | Description |
| --- | --- |
| `New(logger *zap.Logger, timeFormat string, utc bool) iris.Handler` | Logs every request. |
| `NewWithConfig(logger *zap.Logger, conf *Config) iris.Handler` | The same with every option. |

Each request is one log entry with the fields `status`, `method`, `path`, `query`, `ip`, `user-agent`, `latency` and, when a time format is given, `time`. The entry is written at the error level, with the error as its message, when a handler stored an error on the Context (`ctx.SetErr`, `ctx.StopWithError`). Otherwise it is written at the info level with the path as its message.

| `Config` field | Description |
| --- | --- |
| `TimeFormat string` | A `time` package layout, for example `time.RFC3339`. Empty leaves the `time` field out. |
| `UTC bool` | Formats the time in UTC instead of the local zone. |
| `SkipPaths []string` | Request paths that are not logged. |
| `Context Fn` | `func(ctx iris.Context) []zapcore.Field`. Returns extra fields for the entry. It runs after the handlers. |

```go
package main

import (
	"time"

	iriszap "github.com/iris-contrib/middleware/zap"
	"github.com/kataras/iris/v14"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	app := iris.New()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	app.Use(iriszap.NewWithConfig(logger, &iriszap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/health"},
		Context: func(ctx iris.Context) []zapcore.Field {
			// Log the request id that a handler set on the response.
			if id := ctx.ResponseHeader("X-Request-Id"); id != "" {
				return []zapcore.Field{zap.String("request_id", id)}
			}
			return nil
		},
	}))

	app.Get("/health", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	app.Get("/ping", func(ctx iris.Context) {
		ctx.SetHeader("X-Request-Id", "1234-5678-9012")
		ctx.WriteString("pong")
	})

	app.Listen(":8080")
}
```

[_examples/example_2](_examples/example_2/main.go) also logs the request body and the OpenTelemetry trace and span ids. It is a Go module of its own, because it needs the OpenTelemetry packages, which the middleware does not depend on.

## Panic recovery

| Function | Description |
| --- | --- |
| `RecoveryWithZap(logger *zap.Logger, stack bool) iris.Handler` | Recovers, logs the panic at the error level and answers 500 with an empty body. `stack` adds the stack trace to the entry. |
| `CustomRecoveryWithZap(logger, stack, recovery func(iris.Context, interface{})) iris.Handler` | The same, with your own function for the response. |

The entry carries the panic value and the request line and headers, as dumped by `httputil.DumpRequest`. That dump includes the `Authorization` and `Cookie` headers, so treat the log as sensitive.

A broken connection (`broken pipe`, `connection reset by peer`) is logged without a stack trace and nothing is written to the client.

## Examples

- [_examples/example_1](_examples/example_1/main.go)
- [_examples/example_2](_examples/example_2/main.go)
