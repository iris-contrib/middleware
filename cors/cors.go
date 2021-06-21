package cors

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12"
)


// Options is a configuration container to setup the CORS middleware.
type Options struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penalty.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string
	// AllowOriginFunc is a custom function to validate the origin. It take the origin
	// as argument and returns true if allowed or false otherwise. If this option is
	// set, the content of AllowedOrigins is ignored.
	AllowOriginFunc func(origin string) bool
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (HEAD, GET and POST).
	AllowedMethods []string
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [] but "Origin" is always appended to the list.
	AllowedHeaders []string
	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool
	// OptionsPassthrough instructs preflight to let other potential next handlers to
	// process the OPTIONS method. Turn this on if your application handles OPTIONS.
	OptionsPassthrough bool
	// Debugging flag adds additional output to debug server side CORS issues
	Debug bool
}

// Cors http handler
type Cors struct {
	// Debug logger
	Log *log.Logger
	// Normalized list of plain allowed origins
	allowedOrigins []string
	// List of allowed origins containing wildcards
	allowedWOrigins []wildcard
	// Optional origin validator function
	allowOriginFunc func(origin string) bool
	// Normalized list of allowed headers
	allowedHeaders []string
	// Normalized list of allowed methods
	allowedMethods []string
	// Normalized list of exposed headers
	exposedHeaders []string
	maxAge         int
	// Set to true when allowed origins contains a "*"
	allowedOriginsAll bool
	// Set to true when allowed headers contains a "*"
	allowedHeadersAll bool
	allowCredentials  bool
	optionPassthrough bool
}

// New creates a new Cors handler with the provided options.
// Use the Application.UseRouter method to register it globally,
// this is the best option as it enables all the middleware's features.
// Or to register it per group of routes use:
// the Party.AllowMethods(iris.MethodOptions) and Party.Use methods instead.
func New(options Options) iris.Handler {
	c := &Cors{
		exposedHeaders:    convert(options.ExposedHeaders, http.CanonicalHeaderKey),
		allowOriginFunc:   options.AllowOriginFunc,
		allowCredentials:  options.AllowCredentials,
		maxAge:            options.MaxAge,
		optionPassthrough: options.OptionsPassthrough,
	}
	if options.Debug {
		c.Log = log.New(os.Stdout, "[cors] ", log.LstdFlags)
	}

	// Normalize options
	// Note: for origins and methods matching, the spec requires a case-sensitive matching.
	// As it may error prone, we chose to ignore the spec here.

	// Allowed Origins
	if len(options.AllowedOrigins) == 0 {
		if options.AllowOriginFunc == nil {
			// Default is all origins
			c.allowedOriginsAll = true
		}
	} else {
		c.allowedOrigins = []string{}
		c.allowedWOrigins = []wildcard{}
		for _, origin := range options.AllowedOrigins {
			// Normalize
			origin = strings.ToLower(origin)
			if origin == "*" {
				// If "*" is present in the list, turn the whole list into a match all
				c.allowedOriginsAll = true
				c.allowedOrigins = nil
				c.allowedWOrigins = nil
				break
			} else if i := strings.IndexByte(origin, '*'); i >= 0 {
				// Split the origin in two: start and end string without the *
				w := wildcard{origin[0:i], origin[i+1:]}
				c.allowedWOrigins = append(c.allowedWOrigins, w)
			} else {
				c.allowedOrigins = append(c.allowedOrigins, origin)
			}
		}
	}

	// Allowed Headers
	if len(options.AllowedHeaders) == 0 {
		// Use sensible defaults
		c.allowedHeaders = []string{"Origin", "Accept", "Content-Type", "X-Requested-With"}
	} else {
		// Origin is always appended as some browsers will always request for this header at preflight
		c.allowedHeaders = convert(append(options.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		for _, h := range options.AllowedHeaders {
			if h == "*" {
				c.allowedHeadersAll = true
				c.allowedHeaders = nil
				break
			}
		}
	}

	// Allowed Methods
	if len(options.AllowedMethods) == 0 {
		// Default is spec's "simple" methods
		c.allowedMethods = []string{"GET", "POST", "HEAD"}
	} else {
		c.allowedMethods = convert(options.AllowedMethods, strings.ToUpper)
	}

	return c.Serve
}

// Default creates a new Cors handler with default options.
func Default() iris.Handler {
	return New(Options{})
}

// AllowAll create a new Cors handler with permissive configuration allowing all
// origins with all standard methods with any header and credentials.
func AllowAll() iris.Handler {
	return New(Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
}

// Serve apply the CORS specification on the request, and add relevant CORS headers
// as necessary.
func (c *Cors) Serve(ctx iris.Context) {
	if ctx.Method() == http.MethodOptions && ctx.GetHeader("Access-Control-Request-Method") != "" {
		c.logf("Serve: Preflight request")
		c.handlePreflight(ctx)
		if c.optionPassthrough { // handle the options by routes.
			ctx.Next()
			return
		}

		if !ctx.IsStopped() {
			// just 200.
			ctx.StatusCode(http.StatusOK)
			ctx.StopExecution()
		}

		return
	}

	c.logf("Serve: Actual request")
	c.handleActualRequest(ctx)
	ctx.Next()
}

// handlePreflight handles pre-flight CORS requests
func (c *Cors) handlePreflight(ctx iris.Context) {
	origin := ctx.GetHeader("Origin")

	if ctx.Method() != http.MethodOptions {
		c.logf("  Preflight aborted: %s!=OPTIONS", ctx.Method())
		//
		ctx.StatusCode(iris.StatusForbidden)
		ctx.StopExecution()
		//ctx.StopWithStatus(iris.StatusForbidden)
		//
		return
	}
	// Always set Vary headers.
	ctx.Header("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")

	if origin == "" {
		c.logf("  Preflight aborted: empty origin")
		return
	}
	if !c.isOriginAllowed(origin) {
		c.logf("  Preflight aborted: origin '%s' not allowed", origin)
		//
		ctx.StatusCode(iris.StatusForbidden)
		ctx.StopExecution()
		//ctx.StopWithStatus(iris.StatusForbidden)
		//
		return
	}

	reqMethod := ctx.GetHeader("Access-Control-Request-Method")
	if !c.isMethodAllowed(reqMethod) {
		c.logf("  Preflight aborted: method '%s' not allowed", reqMethod)
		//
		ctx.StatusCode(iris.StatusForbidden)
		ctx.StopExecution()
		//ctx.StopWithStatus(iris.StatusForbidden)
		//
		return
	}
	reqHeaders := parseHeaderList(ctx.GetHeader("Access-Control-Request-Headers"))
	if !c.areHeadersAllowed(reqHeaders) {
		c.logf("  Preflight aborted: headers '%v' not allowed", reqHeaders)
		//
		ctx.StatusCode(iris.StatusForbidden)
		ctx.StopExecution()
		//ctx.StopWithStatus(iris.StatusForbidden)
		//
		return
	}
	if c.allowedOriginsAll && !c.allowCredentials {
		ctx.Header("Access-Control-Allow-Origin", "*")
	} else {
		ctx.Header("Access-Control-Allow-Origin", origin)
	}
	// Spec says: Since the list of methods can be unbounded, simply returning the method indicated
	// by Access-Control-Request-Method (if supported) can be enough
	ctx.Header("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
		// Spec says: Since the list of headers can be unbounded, simply returning supported headers
		// from Access-Control-Request-Headers can be enough
		ctx.Header("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if c.allowCredentials {
		ctx.Header("Access-Control-Allow-Credentials", "true")
	}
	if c.maxAge > 0 {
		ctx.Header("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
	}
	c.logf("  Preflight response headers: %v", ctx.ResponseWriter().Header())
}

// handleActualRequest handles simple cross-origin requests, actual request or redirects
func (c *Cors) handleActualRequest(ctx iris.Context) {
	origin := ctx.GetHeader("Origin")

	if ctx.Method() == http.MethodOptions {
		c.logf("  Actual request no headers added: method == %s", ctx.Method())
		//
		ctx.StatusCode(iris.StatusMethodNotAllowed)
		ctx.StopExecution()
		//ctx.StopWithStatus(iris.StatusMethodNotAllowed)
		//
		return
	}
	// Always set Vary, see https://github.com/rs/cors/issues/10
	ctx.ResponseWriter().Header().Add("Vary", "Origin")
	if origin == "" && !c.allowedOriginsAll {
		c.logf("  Actual request no headers added: missing origin")
		return
	}

	if !c.isOriginAllowed(origin) {
		c.logf("  Actual request no headers added: origin '%s' not allowed", origin)
		//
		ctx.StatusCode(iris.StatusForbidden)
		ctx.StopExecution()
		//ctx.StopWithStatus(iris.StatusForbidden)
		//
		return
	}

	// Note that spec does define a way to specifically disallow a simple method like GET or
	// POST. Access-Control-Allow-Methods is only used for pre-flight requests and the
	// spec doesn't instruct to check the allowed methods for simple cross-origin requests.
	// We think it's a nice feature to be able to have control on those methods though.
	if !c.isMethodAllowed(ctx.Method()) {
		c.logf("  Actual request no headers added: method '%s' not allowed", ctx.Method())
		ctx.StatusCode(iris.StatusForbidden)
		ctx.StopExecution()
		//ctx.StopWithStatus(iris.StatusForbidden)
		return
	}
	if c.allowedOriginsAll && !c.allowCredentials {
		ctx.Header("Access-Control-Allow-Origin", "*")
	} else {
		ctx.Header("Access-Control-Allow-Origin", origin)
	}
	if len(c.exposedHeaders) > 0 {
		ctx.Header("Access-Control-Expose-Headers", strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		ctx.Header("Access-Control-Allow-Credentials", "true")
	}
	c.logf("  Actual response added headers: %v", ctx.ResponseWriter().Header())
}

// convenience method. checks if debugging is turned on before printing
func (c *Cors) logf(format string, a ...interface{}) {
	if c.Log != nil {
		c.Log.Printf(format, a...)
	}
}

// isOriginAllowed checks if a given origin is allowed to perform cross-domain requests
// on the endpoint
func (c *Cors) isOriginAllowed(origin string) bool {
	if c.allowOriginFunc != nil {
		return c.allowOriginFunc(origin)
	}
	if c.allowedOriginsAll {
		return true
	}
	origin = strings.ToLower(origin)
	for _, o := range c.allowedOrigins {
		if o == origin {
			return true
		}
	}
	for _, w := range c.allowedWOrigins {
		if w.match(origin) {
			return true
		}
	}
	return false
}

// isMethodAllowed checks if a given method can be used as part of a cross-domain request
// on the endpoing
func (c *Cors) isMethodAllowed(method string) bool {
	if len(c.allowedMethods) == 0 {
		// If no method allowed, always return false, even for preflight request
		return false
	}
	method = strings.ToUpper(method)
	if method == http.MethodOptions {
		// Always allow preflight requests
		return true
	}
	for _, m := range c.allowedMethods {
		if m == method {
			return true
		}
	}
	return false
}

// areHeadersAllowed checks if a given list of headers are allowed to used within
// a cross-domain request.
func (c *Cors) areHeadersAllowed(requestedHeaders []string) bool {
	if c.allowedHeadersAll || len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		header = http.CanonicalHeaderKey(header)
		found := false
		for _, h := range c.allowedHeaders {
			if h == header {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
