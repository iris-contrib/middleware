package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	iriszap "github.com/iris-contrib/middleware/zap"
	"github.com/kataras/iris/v12"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
