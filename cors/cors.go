package cors

import (
	"net/http"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/handlerconv"
	"github.com/kataras/iris/core/router"

	"github.com/rs/cors"
)

// Options is a configuration container to setup the CORS.
// All Options are working EXCEPT on some routers (and the iris' default)
// AllowMethods field is not working.
type Options cors.Options

// MakeFallbackHandler make a fallback handler to add to fallback stack (`app.Fallback(h)`)
// from handler created by `New()` function or `NewAllowAll()` function.
func MakeFallbackHandler(cors_handler context.Handler) context.Handler {
	return func(ctx context.Context) {
		if ctx.Method() != "OPTIONS" {
			ctx.NextOrNotFound()
			return
		}

		uri := ctx.Request().RequestURI
		method := ctx.GetHeader("Access-Control-Request-Method")

		if ctx.RouteExists(method, uri) {
			cors_handler(ctx) // Call the original CORS middleware
			return
		}

		ctx.NextOrNotFound()
	}
}

// New returns a new cors per-route middleware with the provided options.
// Unlike the cors wrapper, this middleware can be registered to specific routes.
func New(opts Options) context.Handler {
	h := handlerconv.FromStdWithNext(WrapNext(opts))
	return h
}

// NewAllowAll returns a new cors per-route middleware with all permissions.
// Unlike the cors wrapper, this middleware can be registered to specific routes.
func NewAllowAll() context.Handler {
	return handlerconv.FromStdWithNext(cors.AllowAll().ServeHTTP)
}

// NewAppMiddleware create a new cors middleware with the provided options to be registered in Iris Application
//   by `app.Configure()` function (`iris.Application#Configure`).
func NewAppMiddleware(opts Options) iris.Configurator {
	return func(app *iris.Application) {
		h := New(opts)

		app.UseGlobal(h)
		app.Fallback(MakeFallbackHandler(h))
	}
}

// NewPartyMiddleware create a new cors middleware with the provided options to be registered in Iris Party
//   by `app.Configure()` function (`iris/core/router.Party#ConfigureParty`).
func NewPartyMiddleware(opts Options) router.PartyConfigurator {
	return func(party router.Party) {
		h := New(opts)

		party.Use(h)
		party.Fallback(MakeFallbackHandler(h))
	}
}

// NewAllowAllAppMiddleware create a new cors middleware with all permissions to be registered in Iris Application
//   by `app.Configure()` function (`iris.Application#Configure`).
func NewAllowAllAppMiddleware() iris.Configurator {
	return func(app *iris.Application) {
		h := NewAllowAll()

		app.UseGlobal(h)
		app.Fallback(MakeFallbackHandler(h))
	}
}

// NewAllowAllPartyMiddleware create a new cors middleware with all permissions to be registered in Iris Party
//   by `app.Configure()` function (`iris/core/router.Party#ConfigureParty`).
func NewAllowAllPartyMiddleware() router.PartyConfigurator {
	return func(party router.Party) {
		h := NewAllowAll()

		party.Use(h)
		party.Fallback(MakeFallbackHandler(h))
	}
}

// NewDefaultAppMiddleware create a new cors middleware with the default settings to be registered in Iris Application
//   by `app.Configure()` function (`iris.Application#Configure`).
func NewDefaultAppMiddleware() iris.Configurator {
	return func(app *iris.Application) {
		h := Default()

		app.UseGlobal(h)
		app.Fallback(MakeFallbackHandler(h))
	}
}

// NewDefaultPartyMiddleware create a new cors middleware with the default settings to be registered in Iris Party
//   by `app.Configure()` function (`iris/core/router.Party#ConfigureParty`).
func NewDefaultPartyMiddleware() router.PartyConfigurator {
	return func(party router.Party) {
		h := Default()

		party.Use(h)
		party.Fallback(MakeFallbackHandler(h))
	}
}

// Default returns a new cors per-route middleware with the default settings:
// allow all origins, allow methods: GET and POST
func Default() context.Handler {
	return New(Options{})
}

// WrapNext is the same as New but it is being used to wrap the entire
// iris' router, even before the method and path matching,
// i.e: app.WrapRouter(WrapNext(Options{...}))
func WrapNext(opts Options) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	h := cors.New(cors.Options(opts)).ServeHTTP
	return h
}
