// This has been cloned to work with iris,
// credits goes to https://github.com/gorilla/csrf,
// I did nothing special here except the performance boost becauase of Iris' ecosystem.

package csrf

import (
	"net/url"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/errors"

	"github.com/gorilla/securecookie"
)

// CSRF token length in bytes.
const tokenLength = 32

// Context/session keys & prefixes
const (
	tokenKey     string = "csrf.Token"
	formKey      string = "csrf.Form"
	errorKey     string = "csrf.Error"
	skipCheckKey string = "csrf.Skip"
	cookieName   string = "_iris_csrf"
	errorPrefix  string = "csrf middleware: "
)

var (
	// The name value used in form fields.
	fieldName = tokenKey
	// defaultAge sets the default MaxAge for cookies.
	defaultAge = 3600 * 12
	// The default HTTP request header to inspect
	headerName = "X-CSRF-Token"
	// Idempotent (safe) methods as defined by RFC7231 section 4.2.2.
	safeMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}
)

// TemplateTag provides a default template tag - e.g. {{ .csrfField }} - for use
// with the TemplateField function.
var TemplateTag = "csrfField"

var (
	// ErrNoReferer is returned when a HTTPS request provides an empty Referer
	// header.
	ErrNoReferer = errors.New("referer not supplied")
	// ErrBadReferer is returned when the scheme & host in the URL do not match
	// the supplied Referer header.
	ErrBadReferer = errors.New("referer invalid")
	// ErrNoToken is returned if no CSRF token is supplied in the request.
	ErrNoToken = errors.New("CSRF token not found in request")
	// ErrBadToken is returned if the CSRF token in the request does not match
	// the token in the session, or is otherwise malformed.
	ErrBadToken = errors.New("CSRF token invalid")
)

// Csrf the middleware container.
type Csrf struct {
	sc   *securecookie.SecureCookie
	st   store
	opts options
}

// options contains the optional settings for the CSRF middleware.
type options struct {
	MaxAge int
	Domain string
	Path   string
	// Note that the function and field names match the case of the associated
	// http.Cookie field instead of the "correct" HTTPOnly name that golint suggests.
	HTTPOnly      bool
	Secure        bool
	RequestHeader string
	FieldName     string
	ErrorHandler  context.Handler
	CookieName    string
}

// New returns a new csrf middleware. It contains both `Get/Head/Options/Trace` and 'Unsafe' methods (i.e `Post`)
// handlers for processing.
func New(authKey []byte, opts ...Option) *Csrf {
	cs := parseOptions(opts...)

	// Set the defaults if no options have been specified
	if cs.opts.ErrorHandler == nil {
		cs.opts.ErrorHandler = unauthorizedHandler
	}

	if cs.opts.MaxAge < 0 {
		// Default of 12 hours
		cs.opts.MaxAge = defaultAge
	}

	if cs.opts.FieldName == "" {
		cs.opts.FieldName = fieldName
	}

	if cs.opts.CookieName == "" {
		cs.opts.CookieName = cookieName
	}

	if cs.opts.RequestHeader == "" {
		cs.opts.RequestHeader = headerName
	}

	// Create an authenticated securecookie instance.
	cs.sc = securecookie.New(authKey, nil)
	// Use JSON serialization (faster than one-off gob encoding)
	cs.sc.SetSerializer(securecookie.JSONEncoder{})
	// Set the MaxAge of the underlying securecookie.
	cs.sc.MaxAge(cs.opts.MaxAge)

	// Default to the cookieStore
	cs.st = &cookieStore{
		name:     cs.opts.CookieName,
		maxAge:   cs.opts.MaxAge,
		secure:   cs.opts.Secure,
		httpOnly: cs.opts.HTTPOnly,
		path:     cs.opts.Path,
		domain:   cs.opts.Domain,
		sc:       cs.sc,
	}

	return cs
}

// Protect is HTTP middleware that provides Cross-Site Request Forgery
// protection.
//
// It securely generates a masked (unique-per-request) token that
// can be embedded in the HTTP response (e.g. form field or HTTP header).
// The original (unmasked) token is stored in the session, which is inaccessible
// by an attacker (provided you are using HTTPS). Subsequent requests are
// expected to include this token, which is compared against the session token.
// Requests that do not provide a matching token are served with a HTTP 403
// 'Forbidden' error response.
//
// Example: https://github.com/iris-contrib/middleware/tree/master/csrf/_example
func Protect(authKey []byte, opts ...Option) context.Handler {
	cs := New(authKey, opts...)
	return cs.Serve
}

// Serve implements iris.Handler for the csrf type.
func (cs *Csrf) Serve(ctx context.Context) {
	// Skip the check if directed to. This should always be a bool.
	if skip, _ := ctx.Values().GetBool(skipCheckKey); skip {
		ctx.Next()
		return
	}

	// Retrieve the token from the session.
	// An error represents either a cookie that failed HMAC validation
	// or that doesn't exist.
	realToken, err := cs.st.Get(ctx)

	if err != nil || len(realToken) != tokenLength {
		// If there was an error retrieving the token, the token doesn't exist
		// yet, or it's the wrong length, generate a new token.
		// Note that the new token will (correctly) fail validation downstream
		// as it will no longer match the request token.
		realToken, err = generateRandomBytes(tokenLength)
		if err != nil {
			envError(ctx, err)
			cs.opts.ErrorHandler(ctx)
			return
		}

		// Save the new (real) token in the session store.
		err = cs.st.Save(ctx, realToken)
		if err != nil {
			envError(ctx, err)
			cs.opts.ErrorHandler(ctx)
			return
		}
	}

	// Save the masked token to the request context
	ctx.Values().Set(tokenKey, mask(realToken))
	// Save the field name to the request context
	ctx.Values().Set(formKey, cs.opts.FieldName)

	// HTTP methods not defined as idempotent ("safe") under RFC7231 require
	// inspection.
	if !contains(safeMethods, ctx.Method()) {

		r := ctx.Request()
		// Enforce an origin check for HTTPS connections. As per the Django CSRF
		// implementation (https://goo.gl/vKA7GE) the Referer header is almost
		// always present for same-domain HTTP requests.
		if r.URL.Scheme == "https" {
			// Fetch the Referer value. Call the error handler if it's empty or
			// otherwise fails to parse.
			referer, err := url.Parse(r.Referer())
			if err != nil || referer.String() == "" {
				envError(ctx, ErrNoReferer)
				cs.opts.ErrorHandler(ctx)
				return
			}

			if sameOrigin(r.URL, referer) == false {
				envError(ctx, ErrBadReferer)
				cs.opts.ErrorHandler(ctx)
				return
			}
		}

		// If the token returned from the session store is nil for non-idempotent
		// ("unsafe") methods, call the error handler.
		if realToken == nil {
			envError(ctx, ErrNoToken)
			cs.opts.ErrorHandler(ctx)
			return
		}

		// Retrieve the combined token (pad + masked) token and unmask it.
		requestToken := unmask(cs.requestToken(r))

		// Compare the request token against the real token
		if !compareTokens(requestToken, realToken) {
			envError(ctx, ErrBadToken)
			cs.opts.ErrorHandler(ctx)
			return
		}

	}

	// Set the Vary: Cookie header to protect clients from caching the response.
	ctx.Header("Vary", "Cookie")
	// Call the wrapped next handler on success.
	ctx.Next()
}

// unauthorizedhandler sets a HTTP 403 Forbidden status and writes the
// CSRF failure reason to the response.
func unauthorizedHandler(ctx context.Context) {
	ctx.StatusCode(iris.StatusForbidden)
	err := FailureReason(ctx)
	if err != nil {
		ctx.WriteString(err.Error())
	}
	return
}
