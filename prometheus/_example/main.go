package main

import (
	"math/rand"
	"time"

	"github.com/kataras/iris/v14"

	prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	app := iris.New()
	m := prometheusMiddleware.New("serviceName", 0.3, 1.2, 5.0)

	// Use on the root application records every request, the ones that
	// match no route (404, 405) included. Do not call m.ServeHTTP again
	// from a HandleNoRoute handler: that would count a not found twice.
	app.Use(m.ServeHTTP)

	app.Get("/", func(ctx iris.Context) {
		sleep := rand.Intn(4999) + 1
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		ctx.Writef("Slept for %d milliseconds", sleep)
	})

	app.Get("/metrics", iris.FromStd(promhttp.Handler()))

	// http://localhost:8080/
	// http://localhost:8080/anotfound
	// http://localhost:8080/metrics
	app.Listen(":8080")
}
