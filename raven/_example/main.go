package main

import (
	"github.com/kataras/iris/v12"

	// $ go get github.com/getsentry/raven-go
	"github.com/getsentry/raven-go"
	// $ go get github.com/iris-contrib/middleware/raven
	ravenIris "github.com/iris-contrib/middleware/raven"
)

// https://docs.sentry.io/clients/go/integrations/http/
func init() {
	raven.SetDSN("https://<key>:<secret>@sentry.io/<project>")
}

func main() {
	app := iris.New()
	app.Use(ravenIris.RecoveryHandler)

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hi")
	})

	app.Listen(":8080")
}
