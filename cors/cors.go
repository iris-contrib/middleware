package cors

//  +------------------------------------------------------------+
//  | Middleware usage                                           |
//  +------------------------------------------------------------+
//
// import (
//  "gopkg.in/kataras/iris.v6"
//  "gopkg.in/kataras/iris.v6/adaptors/httprouter"
//  "github.com/iris-contrib/middleware/cors"
// )
//
// app := iris.New()
// app.Adapt(httprouter.New())
// app.Post("/user", cors.Default(), func(ctx *iris.Context){})
// app.Listen(":8080")

import (
	"github.com/rs/cors"
	"gopkg.in/kataras/iris.v6"
)

// Options is a configuration container to setup the CORS.
// All Options are working EXCEPT on some routers (and the iris' default)
// AllowMethods field is not working.
type Options cors.Options

// New returns a new cors per-route middleware
// with the provided options.
// Unlike the cors wrapper, this middleware can be registered to specific routes,
// Options.AllowedMethods is missing
func New(opts Options) iris.HandlerFunc {
	return iris.ToHandler(cors.New(cors.Options(opts)).ServeHTTP)
}

// Default returns a new cors per-route middleware with the default settings:
// allow all origins, allow methods: GET and POST
func Default() iris.HandlerFunc {
	return New(Options{})
}
