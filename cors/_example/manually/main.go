package main

import "github.com/kataras/iris/v14"

func main() {
	app := iris.New()

	crs := func(ctx iris.Context) {
		ctx.SetHeaderOverride("Access-Control-Allow-Origin", "*")
		ctx.SetHeaderOverride("Access-Control-Allow-Credentials", "true")

		if ctx.Method() == iris.MethodOptions {
			ctx.SetHeaderOverride("Access-Control-Methods", "POST, PUT, PATCH, DELETE")
			ctx.SetHeaderOverride("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type")
			ctx.SetHeaderOverride("Access-Control-Max-Age", "86400")
			ctx.StatusCode(iris.StatusNoContent)
			return
		}

		ctx.Next()
	} // or	"github.com/iris-contrib/middleware/cors"

	app.UseRouter(crs)
	// OR per group of routes:
	// api := app.Router("/api")
	// api.AllowMethods(iris.MethodOptions) <- important for the preflight.
	// api.Use(crs)

	api := app.Router("/api")
	api.Post("/mailer", func(ctx iris.Context) {
		var any iris.Map
		err := ctx.ReadJSON(&any)
		if err != nil {
			ctx.StopWithError(iris.StatusBadRequest, err)
			return
		}
		ctx.Application().Logger().Infof("received %#+v", any)

		ctx.JSON(iris.Map{"message": "ok"})
	})

	api.Post("/send", func(ctx iris.Context) {
		ctx.WriteString("sent")
	})
	api.Put("/send", func(ctx iris.Context) {
		ctx.WriteString("updated")
	})
	api.Delete("/send", func(ctx iris.Context) {
		ctx.WriteString("deleted")
	})

	app.Listen(":8080")
}
