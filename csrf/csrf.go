package csrf

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/csrf.*", "iris-contrib.csrf.token")
}

// CSRF token length in bytes.
const tokenLength = 32

// Context/session keys.
const (
	tokenKey          string = "csrf.token"
	formKey           string = "csrf.Form"
	skipCheckKey      string = "csrf.Skip"
	DefaultCookieName string = "_iris_csrf"
)

var (
	// DefaultFieldName is the default name value used in form fields.
	DefaultFieldName = tokenKey
	// DefaultSameSite sets the `SameSite` cookie attribute, which is
	// invalid in some older browsers due to changes in the SameSite spec. These
	// browsers will not send the cookie to the server.
	// The csrf middleware uses the http.SameSiteLaxMode (SameSite=Lax) as the default one.
	DefaultSameSite = http.SameSiteLaxMode
	// DefaultMaxAge sets the default MaxAge for cookies.
	DefaultMaxAge = 3600 * 12
	// DefaultRequestHeader is the default HTTP request header to inspect.
	DefaultRequestHeader = "X-CSRF-Token"
	// TemplateTag is the default HTML template tag provides a default template tag,
	// e.g. {{ .csrfField }} - for use
	// with the TemplateField function.
	TemplateTag = "csrfField"
	// Idempotent (safe) methods as defined by RFC7231 section 4.2.2.
	safeMethods = []string{iris.MethodGet, iris.MethodHead, iris.MethodOptions, iris.MethodTrace}
)

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

// CSRF represents the CSRF feature.
type CSRF struct {
	opts *Options
}

// New returns the CSRF middleware.
// Read the `Protect` package-level function for more.
func New(opts Options) *CSRF {
	if opts.Store == nil {
		panic("Store is required")
	}

	if opts.RequestHeader == "" {
		opts.RequestHeader = DefaultRequestHeader
	}

	if opts.FieldName == "" {
		opts.FieldName = DefaultFieldName
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = UnauthorizedHandler
	}

	return &CSRF{opts: &opts}
}

// Protect is Iris middleware that provides Cross-Site Request Forgery
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
// See `New` package-level function for further customizations.
func Protect(authKey []byte, cookieOpts ...CookieOption) iris.Handler {
	csrf := New(Options{
		Store: NewCookieStore(authKey, cookieOpts...),
	})

	return csrf.Protect
}

// Protect is Iris middleware that provides Cross-Site Request Forgery
// protection.
//
// Read more at the Protect package-level function's documentation.
func (csrf *CSRF) Protect(ctx iris.Context) {
	if csrf.Filter(ctx) {
		// Call the next handler in the chain on success.
		ctx.Next()
	} else {
		csrf.opts.ErrorHandler(ctx)
	}
}

// Filter is the Iris Filter type of the csrf middleware.
// It can be used instead of the `Protect` method when
// the caller needs to manually manage the response on
// success (true) and failure (false). The `ErrorHandler` is not fired.
func (csrf *CSRF) Filter(ctx iris.Context) bool {
	opts := csrf.opts

	// Skip the check if directed to. This should always be a bool.
	if val, err := contextGet(ctx, skipCheckKey); err == nil {
		if skip, ok := val.(bool); ok {
			if skip {
				return true
			}
		}
	}

	// Retrieve the token from the session.
	// An error represents either a cookie that failed HMAC validation
	// or that doesn't exist.
	realToken, err := opts.Store.Get(ctx)
	if err != nil || len(realToken) != tokenLength {
		// If there was an error retrieving the token, the token doesn't exist
		// yet, or it's the wrong length, generate a new token.
		// Note that the new token will (correctly) fail validation downstream
		// as it will no longer match the request token.
		realToken, err = generateRandomBytes(tokenLength)
		if err != nil {
			envError(ctx, err)
			return false
		}

		// Save the new (real) token in the session store.
		err = opts.Store.Save(ctx, realToken)
		if err != nil {
			envError(ctx, err)
			return false
		}
	}

	// Save the masked token to the request context
	contextSave(ctx, tokenKey, mask(realToken))
	// Save the field name to the request context in order for TemplateField to work.
	contextSave(ctx, formKey, opts.FieldName)

	// HTTP methods not defined as idempotent ("safe") under RFC7231 require
	// inspection.
	if !contains(safeMethods, ctx.Method()) {
		// Enforce an origin check for HTTPS connections. As per the Django CSRF
		// implementation (https://goo.gl/vKA7GE) the Referer header is almost
		// always present for same-domain HTTP requests.
		if ctx.Scheme() == "https://" {
			// Fetch the Referer value. Call the error handler if it's empty or
			// otherwise fails to parse.

			referer, err := url.Parse(ctx.GetReferrer().String())
			if err != nil || referer.String() == "" {
				envError(ctx, ErrNoReferer)
				return false
			}

			if !strings.EqualFold(referer.Host, ctx.Host()) {
				if !contains(opts.TrustedOrigins, referer.Host) {
					envError(ctx, ErrBadReferer)
					return false
				}
			}
		}

		// If the token returned from the session store is nil for non-idempotent
		// ("unsafe") methods, call the error handler.
		if realToken == nil {
			envError(ctx, ErrNoToken)
			return false
		}

		// Retrieve the combined token (pad + masked) token and unmask it.
		requestToken := unmask(csrf.RequestToken(ctx))

		// Compare the request token against the real token
		if !compareTokens(requestToken, realToken) {
			envError(ctx, ErrBadToken)
			return false
		}
	}

	// Set the Vary: Cookie header to protect clients from caching the response.
	ctx.Header("Vary", "Cookie")
	return true
}

// RequestToken returns the issued token (pad + masked token) from the HTTP POST
// body or HTTP header. It will return nil if the token fails to decode.
func (csrf *CSRF) RequestToken(ctx iris.Context) []byte {
	// 1. Check the HTTP header first.
	issued := ctx.GetHeader(csrf.opts.RequestHeader)

	// 2. Fall back to the POST (form) value.
	if issued == "" {
		issued = ctx.PostValue(csrf.opts.FieldName)
	}

	// 3. Finally, fall back to the multipart form (if set).
	if issued == "" {
		issued = ctx.FormValue(csrf.opts.FieldName)
	}

	// Decode the "issued" (pad + masked) token sent in the request. Return a
	// nil byte slice on a decoding error (this will fail upstream).
	decoded, err := base64.StdEncoding.DecodeString(issued)
	if err != nil {
		return nil
	}

	return decoded
}

// UnauthorizedHandler sets a HTTP 403 Forbidden status and writes the
// CSRF failure reason to the response.
func UnauthorizedHandler(ctx iris.Context) {
	ctx.StopWithStatus(iris.StatusForbidden)
}
