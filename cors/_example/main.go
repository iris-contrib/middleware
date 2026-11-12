package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/cors"
)

func main() {
	app := iris.New()

	crs := cors.New(cors.Options{
		// The origins that may call this API from a browser: here the page
		// that the client example serves. "*" allows every origin; do not
		// combine it with AllowCredentials.
		AllowedOrigins:   []string{"http://localhost:9090"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})

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
