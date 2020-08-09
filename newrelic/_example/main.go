package main

import (
	"os"
	"time"

	"github.com/kataras/iris/v12"

	irisnewrelic "github.com/iris-contrib/middleware/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	app := iris.New()

	// Create a newrelic.Application and return the middleware.
	m, err := irisnewrelic.New(
		newrelic.ConfigAppName("My Application"),
		newrelic.ConfigLicense(os.Getenv("NEWRELIC_LIECNSE_KEY")),
	)
	if err != nil {
		app.Logger().Fatal(err)
	}

	// Or wrap an existing newrelic Application and return the middleware:
	// m := irisnewrelic.Wrap(existingApplication)

	// Register the middleware.
	app.Use(m)

	app.Get("/", func(ctx iris.Context) {
		// The txn.End() method will be AUTOMATICALLY called
		// at the end of the handler chain of this request,
		// no need to call it by yourself.
		txn := irisnewrelic.GetTransaction(ctx)

		if v, _ := ctx.URLParamBool("segments"); v {
			// Starts a new segment:
			defer txn.StartSegment("f1").End()

			ctx.WriteString("segments!")
			time.Sleep(8 * time.Millisecond)
		}

		ctx.HTML(`<h3 style="color:green;">%s</h3>`, "success")
	})

	app.Listen(":8080")
}
