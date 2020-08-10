package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("index.html")
	})

	// Serves index.html comments.
	// Navigate to http://localhost:9090,
	// this will act as a client for your server.
	// Don't forget to EDIT the index.html's host variable
	// to match the server's one.
	app.Listen(":9090")
}
