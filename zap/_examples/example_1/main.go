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
