package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/expmetric"
)

func main() {
	// Initialization of the Iris web application.
	app := iris.New()
	// Register the handler that lists the expvars.
	//
	// Tip: you can protect this endpoint with basic authentication.
	app.Get("/debug/vars", expmetric.Handler())

	// Use on the root application counts every request,
	// the ones that match no route (404, 405) included.
	// The available counters are:
	// HitsTotal, HitsPerSecond, HitsPerMinute and HitsPerHour.
	app.Use(expmetric.HitsTotal())

	// To count the requests of matched routes only:
	// app.UseRelevant(expmetric.HitsTotal())
	//
	// To count the requests of a group of routes:
	// usersRouter := app.Router("/users")
	// usersRouter.Use(expmetric.HitsPerMinute())

	// To capture hits on specific endpoint just
	// add it before the main route's handler:
	app.Get("/", expmetric.HitsPerMinute(expmetric.MetricName("hits_minute")), helloWorld)

	// Navigate to http://localhost:8080 and
	// then to http://localhost:8080/debug/vars
	app.Listen(":8080")
}

func helloWorld(ctx iris.Context) {
	ctx.WriteString("Hello, World!")
}
