package cors

import (
	"net/http"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/handlerconv"

	"github.com/rs/cors"
)

// Options is a configuration container to setup the CORS.
// All Options are working EXCEPT on some routers (and the iris' default)
// AllowMethods field is not working.
type Options cors.Options

// New returns a new cors per-route middleware
// with the provided options.
// Unlike the cors wrapper, this middleware can be registered to specific routes,
// Options.AllowedMethods is missing
func New(opts Options) context.Handler {
	h := handlerconv.FromStdWithNext(WrapNext(opts))
	return h
}

// Default returns a new cors per-route middleware with the default settings:
// allow all origins, allow methods: GET and POST
func Default() context.Handler {
	return New(Options{})
}

// WrapNext is the same as New but it is being used to wrap the entire
// Iris' router, even before the method and path matching,
// i.e: app.WrapRouter(WrapNext(Options{...}))
func WrapNext(opts Options) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	h := cors.New(cors.Options(opts)).ServeHTTP
	return h
}
