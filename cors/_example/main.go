package main

import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/cors"
)

func main() {
	app := iris.New()

	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // allows everything, use that to change the hosts.
		AllowCredentials: true,
	})

	app.UseRouter(crs)
	// OR per group of routes:
	// api := app.Party("/api")
	// api.AllowMethods(iris.MethodOptions) <- important for the preflight.
	// api.Use(crs)

	api := app.Party("/api")
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

	app.Listen(":8080", iris.WithTunneling)
}
