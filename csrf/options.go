package csrf

import (
	"net/http"

	"github.com/kataras/iris/v12"
)

// Options describes the configuration for the CRSF middleware.
type Options struct {
	// Store lets you configure the backend storage of the CRSF session.
	Store Store
	// FieldName allows you to change the name attribute of the hidden <input> field
	// inspected by this package. The default is 'csrf.token'.
	FieldName string
	// RequestHeader allows you to change the request header the CSRF middleware
	// inspects. The default is X-CSRF-Token.
	RequestHeader string
	// ErrorHandler allows you to change the handler called when CSRF request
	// processing encounters an invalid token or request. A typical use would be to
	// provide a handler that returns a static HTML file with a HTTP 403 status. By
	// default a HTTP 403 status and a plain text CSRF failure reason are served.
	//
	// Note that a custom error handler can also access the csrf.FailureReason(r)
	// function to retrieve the CSRF validation reason from the request context.
	ErrorHandler iris.Handler
	// TrustedOrigins configures a set of origins (Referers) that are considered as trusted.
	// This will allow cross-domain CSRF use-cases - e.g. where the front-end is served
	// from a different domain than the API server - to correctly pass a CSRF check.
	//
	// You should only provide origins you own or have full control over.
	TrustedOrigins []string
}

// CookieOption represents the Cookie configuration,
// used by CookieStore.
type CookieOption func(*http.Cookie)

// CookieName changes the name of the CSRF cookie issued to clients.
//
// Note that cookie names should not contain whitespace, commas, semicolons,
// backslashes or control characters as per RFC6265.
func CookieName(name string) CookieOption {
	return func(c *http.Cookie) {
		c.Name = name
	}
}

// Secure sets the 'Secure' flag on the cookie. Defaults to true (recommended).
// Set this to 'false' in your development environment otherwise the cookie won't
// be sent over an insecure channel. Setting this via the presence of a 'DEV'
// environmental variable is a good way of making sure this won't make it to a
// production environment.
func Secure(secure bool) CookieOption {
	return func(c *http.Cookie) {
		c.Secure = secure
	}
}

// HTTPOnly sets the 'HttpOnly' flag on the cookie. Defaults to true (recommended).
func HTTPOnly(httpOnly bool) CookieOption {
	return func(c *http.Cookie) {
		c.HttpOnly = httpOnly
	}
}

// SameSite sets the cookie SameSite attribute. Defaults to blank to maintain
// backwards compatibility, however, Strict is recommended.
//
// SameSite(SameSiteStrictMode) will prevent the cookie from being sent by the
// browser to the target site in all cross-site browsing context, even when
// following a regular link (GET request).
//
// SameSite(SameSiteLaxMode) provides a reasonable balance between security and
// usability for websites that want to maintain user's logged-in session after
// the user arrives from an external link. The session cookie would be allowed
// when following a regular link from an external website while blocking it in
// CSRF-prone request methods (e.g. POST).
func SameSite(sameSite http.SameSite) CookieOption {
	return func(c *http.Cookie) {
		c.SameSite = sameSite
	}
}

// MaxAge sets the maximum age (in seconds) of a CSRF token's underlying cookie.
// Defaults to 12 hours. Call csrf.MaxAge(0) to explicitly set session-only
// cookies.
func MaxAge(maxAge int) CookieOption {
	return func(c *http.Cookie) {
		c.MaxAge = maxAge
	}
}

// Domain sets the cookie domain. Defaults to the current domain of the request
// only (recommended).
//
// This should be a hostname and not a URL. If set, the domain is treated as
// being prefixed with a '.' - e.g. "example.com" becomes ".example.com" and
// matches "www.example.com" and "secure.example.com".
func Domain(domain string) CookieOption {
	return func(c *http.Cookie) {
		c.Domain = domain
	}
}

// Path sets the cookie path. Defaults to the path the cookie was issued from
// (recommended).
//
// This instructs clients to only respond with cookie for that path and its
// subpaths - i.e. a cookie issued from "/register" would be included in requests
// to "/register/step2" and "/register/submit".
func Path(path string) CookieOption {
	return func(c *http.Cookie) {
		c.Path = path
	}
}
