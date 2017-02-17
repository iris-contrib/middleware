package main

import (
	// look the https://github.com/kataras/iris/blob/v6/adaptors/cors/_example/main.go
	// if you want support for the cors' allowed methods.
	"github.com/iris-contrib/middleware/cors"
	"gopkg.in/kataras/iris.v6"
)

func main() {

	app := iris.New()
	app.Adapt(iris.DevLogger())
	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	v1 := app.Party("/api/v1")
	v1.Use(crs)
	{
		v1.Post("/home", func(c *iris.Context) {
			app.Log(iris.DevMode, "lalala")
			c.WriteString("Hello from /home")
		})
		v1.Get("/g", func(c *iris.Context) {
			app.Log(iris.DevMode, "lalala")
			c.WriteString("Hello from /home")
		})
		v1.Post("/h", func(c *iris.Context) {
			app.Log(iris.DevMode, "lalala")
			c.WriteString("Hello from /home")
		})
	}

	app.Listen(":8084")
}
