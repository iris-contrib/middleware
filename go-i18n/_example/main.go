package main

import (
	"github.com/kataras/iris/v12"

	i18n "github.com/iris-contrib/middleware/go-i18n"
)

type user struct {
	Name string
}

func main() {
	app := iris.New()
	app.I18n.Reset(i18n.NewLoader("./locales/*.yaml"))
	app.I18n.SetDefault("en-US")

	app.Get("/", func(ctx iris.Context) {
		hi := ctx.Tr("hi", &user{
			Name: "John Doe",
		})

		ctx.Writef("From the language %s translated output: %s", ctx.GetLocale().Language(), hi)
	})

	// http://localhost:8080
	// http://localhost:8080/?lang=zh
	// http://localhost:8080/?lang=zh-CN
	// http://localhost:8080/en
	// http://localhost:8080/zh
	app.Run(iris.Addr(":8080"))
}
