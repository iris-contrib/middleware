package main

import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/expmetric"
)

func main() {
	// Initialization of the Iris web application.
	app := iris.New()
	// To capture all hits (including not found pages and all routes)
	// register the middleware using the UseRouter wrapper like this:
	// app.UseRouter(expmetric.HitsPerMinute())

	// Rregister the handler to list the expvars.
	//
	// Note: Register it after the middleware if you want to capture
	// the hits of the /debug/vars page as well.
	//
	// Tip: you can protect this endpoint with basic authentication.
	app.Get("/debug/vars", expmetric.Handler())

	// app.Get("/", expmetric.HitsPerMinute(), index)

	// To capture hits per group of routes:
	// usersRouter := app.Party("/users")
	// usersRouter.Use(expmetric.HitsPerMinute())

	// To register the middleware only on the below routes (e.g. / -> index):
	// app.Use(expmetric.HitsPerMinute())
	// 				     HitsPerSecond
	// 				     HitsPerHour
	//				     HitsTotal

	app.Use(expmetric.HitsTotal())

	// To capture hits on specific endpoint just
	// add it before the main route's handler:
	app.Get("/", expmetric.HitsPerMinute(expmetric.MetricName("hits_minute")), helloWorld)

	// More options and listening.
	//
	// Navigate to http://localhost:8080 and
	// then to http://localhost:8080/debug/vars
	app.Listen(":8080")
}

func helloWorld(ctx iris.Context) {
	ctx.WriteString("Hello, World!")
}
