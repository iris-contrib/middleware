package cors

import (
	"github.com/kataras/iris"
	"strings"
)

const (
	// AllowAll allow all origins to have access
	AllowAll = "*"
)

// Cors simple cors middleware, use github.com/iris-contrib/plugin/cors for more options.
type Cors struct {
	origins []string
	// allowCredentials turn to true to allow cradentials
	allowCredentials bool
}

// AddOrigin adds an origin to the allowed origins
func (c *Cors) AddOrigin(origin string) {
	c.origins = append(c.origins, origin)
}

// AllowCredentials call it to enable credentials headers.
func (c *Cors) AllowCredentials() {
	c.allowCredentials = true
}

func (c *Cors) originAllowed(origin string) bool {
	origin = strings.ToLower(origin)
	for _, s := range c.origins {
		if s == origin || s == AllowAll {
			return true
		}
	}
	return false
}

// Serve implements the iris.Handler
func (c *Cors) Serve(ctx *iris.Context) {
	// set Vary Origin always as specs says.
	headers := ctx.ResponseWriter.Header()
	headers.Add("Vary", "Origin")

	origin := ctx.RequestHeader("Origin")
	if c.originAllowed(origin) {
		// pre-flight
		if ctx.Method() == iris.MethodOptions {
			headers.Add("Vary", "Access-Control-Request-Method")
			headers.Add("Vary", "Access-Control-Request-Headers")
			headers.Set("Access-Control-Allow-Methods", strings.ToUpper(ctx.Method()))
			// here or return or continue, we will continue with ctx.Next(), pass via Options is acceptable.
		} else {
			if c.allowCredentials {
				headers.Set("Access-Control-Allow-Credentials", "true")
			}
		}
		headers.Set("Access-Control-Allow-Origin", origin)
		ctx.Next()
	}

}

// Conflicts to allow "OPTIONS"
func (c *Cors) Conflicts() string {
	return "httpmethod"
}

// New returns a new cors plugin, allowed origins is optional parameter,
// if not setted then use c.AddOrigin otherwise you have no reason to use this middleware.
func New(origins ...string) *Cors {
	// no let's keep origins empty.
	// if len(origins) == 0 {
	// 	origins = []string{AllowAll}
	// }
	//

	// note: return a *Cors instead of iris.HandlerFunc because
	// the router checks of Conflicts() string in order to by-pass the "OPTIONS" method tree.
	return &Cors{
		origins:          origins,
		allowCredentials: false,
	}
}

// Default returns a new cors plugin, allow all origins but dissalow credentials.
func Default() *Cors {
	return New(AllowAll)
}
