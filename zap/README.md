# zap

Alternative logging through [zap](https://github.com/uber-go/zap) after the [#2125 feature request](https://github.com/kataras/iris/issues/2125). A clone of https://github.com/gin-contrib/zap.

## Usage

### Start using it

Download and install it:

```sh
go get github.com/iris-contrib/middleware/zap@master
```

Import it in your code:

```go
import "github.com/iris-contrib/middleware/zap"
```

## Example

See the [example](_examples/example_1main.go).

```go
package main

import (
	"fmt"
	"time"

	iriszap "github.com/iris-contrib/middleware/zap"
	"github.com/kataras/iris/v12"
	"go.uber.org/zap"
)

func main() {
	app := iris.New()

	logger, _ := zap.NewProduction()

	// Add a iriszap middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	//   - RFC3339 with UTC time format.
	app.Use(iriszap.New(logger, time.RFC3339, true))

	// Logs all panic to error log
	//   - stack means whether output the stack info.
	app.Use(iriszap.RecoveryWithZap(logger, true))

	// Example ping request.
	app.Get("/ping", func(ctx iris.Context) {
		ctx.Text("pong " + fmt.Sprint(time.Now().Unix()))
	})

	// Example when panic happen.
	app.Get("/panic", func(ctx iris.Context) {
		panic("An unexpected error happen!")
	})

	// Listen and Server in 0.0.0.0:8080
	app.Listen(":8080")
}
```

## Skip logging

When you want to skip logging for specific path,
please use `NewWithConfig`

```go

app.Use(NewWithConfig(utcLogger, &Config{
  TimeFormat: time.RFC3339,
  UTC: true,
  SkipPaths: []string{"/no_log"},
}))
```

## Custom Zap fields

Example for custom log request body, response request ID or log [Open Telemetry](https://opentelemetry.io/) TraceID.

```go
func main() {
	app := iris.New()

	logger, _ := zap.NewProduction()

	app.Use(iriszap.NewWithConfig(logger, &iriszap.Config{
		UTC:        true,
		TimeFormat: time.RFC3339,
		Context: iriszap.Fn(func(ctx iris.Context) []zapcore.Field {
			req := ctx.Request()

			fields := []zapcore.Field{}
			// log request ID
			if requestID := ctx.ResponseWriter().Header().Get("X-Request-Id"); requestID != "" {
				fields = append(fields, zap.String("request_id", requestID))
			}

			// log trace and span ID
			if trace.SpanFromContext(req.Context()).SpanContext().IsValid() {
				fields = append(fields, zap.String("trace_id", trace.SpanFromContext(req.Context()).SpanContext().TraceID().String()))
				fields = append(fields, zap.String("span_id", trace.SpanFromContext(req.Context()).SpanContext().SpanID().String()))
			}

			// log request body
			var body []byte
			var buf bytes.Buffer
			tee := io.TeeReader(req.Body, &buf)
			body, _ = io.ReadAll(tee)
			req.Body = io.NopCloser(&buf)
			fields = append(fields, zap.String("body", string(body)))

			return fields
		}),
	}))

	// Example ping request.
	app.Get("/ping", func(ctx iris.Context) {
		ctx.Header("X-Request-Id", "1234-5678-9012")
		ctx.Text("pong " + fmt.Sprint(time.Now().Unix()))
	})

	app.Post("/ping", func(ctx iris.Context) {
		ctx.Header("X-Request-Id", "9012-5678-1234")
		ctx.Text("pong " + fmt.Sprint(time.Now().Unix()))
	})

	// Listen and Server in 0.0.0.0:8080
	app.Listen(":8080")
}
```
