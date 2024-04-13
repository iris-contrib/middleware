package csrf

import (
	"net/http"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/gorilla/securecookie"
)

// Store represents the session storage used for CSRF tokens.
type Store interface {
	// Get returns the real CSRF token from the store.
	Get(ctx iris.Context) ([]byte, error)
	// Save stores the real CSRF token in the store and writes a
	// cookie to the http.ResponseWriter.
	// For non-cookie stores, the cookie should contain a unique (256 bit) ID
	// or key that references the token in the backend store.
	// csrf.GenerateRandomBytes is a helper function for generating secure IDs.
	Save(ctx iris.Context, token []byte) error
}

// cookieStore is a signed cookie session store for CSRF tokens.
type cookieStore struct {
	// The authentication key provided should be 32 bytes
	// long and persist across application restarts.
	authKey []byte
	options http.Cookie
}

// NewCookieStore returns a new Store that saves and retrieves
// the CSRF token to and from the client's cookie jar.
func NewCookieStore(authKey []byte, cookieOpts ...CookieOption) Store {
	opts := http.Cookie{
		Name:     DefaultCookieName,
		Secure:   true,
		HttpOnly: true,
		SameSite: DefaultSameSite,
		MaxAge:   DefaultMaxAge,
	}

	for _, opt := range cookieOpts {
		if opt != nil {
			opt(&opts)
		}
	}

	return &cookieStore{
		authKey: authKey,
		options: opts,
	}
}

var _ Store = (*cookieStore)(nil)

// Note that, normally it would be safe to used across multiple requests as all fields EXCEPT one, the "error"
// is not touched. So... create a new instance on each incoming request.
func (cs *cookieStore) newSecureCookie() *securecookie.SecureCookie {
	if len(cs.authKey) == 0 { // Check if we have an actual key.
		return nil //  Otherwise don't encode/decode the cookie at all (not recommended but exists as an option).
	}

	secureCookie := securecookie.New(cs.authKey, nil)
	// Use JSON serialization (faster than one-off gob encoding)
	secureCookie.SetSerializer(securecookie.JSONEncoder{})
	// Set the MaxAge of the underlying securecookie.
	secureCookie.MaxAge(cs.options.MaxAge)

	return secureCookie
}

// Get retrieves a CSRF token from the session cookie. It returns an empty token
// if decoding fails (e.g. HMAC validation fails or the named cookie doesn't exist).
func (cs *cookieStore) Get(ctx iris.Context) ([]byte, error) {
	// Retrieve the cookie from the request
	cookie, err := ctx.Request().Cookie(cs.options.Name)
	if err != nil {
		return nil, err
	}

	if sc := cs.newSecureCookie(); sc != nil {
		token := make([]byte, tokenLength)
		// Decode the HMAC authenticated cookie.
		err = sc.Decode(cs.options.Name, cookie.Value, &token)
		if err != nil {
			return nil, err
		}

		return token, nil
	}

	return []byte(cookie.Value), nil
}

// Save stores the CSRF token in the session cookie.
func (cs *cookieStore) Save(ctx iris.Context, token []byte) (err error) {
	var value string

	if sc := cs.newSecureCookie(); sc != nil {
		// Generate an encoded cookie value with the CSRF token.
		value, err = sc.Encode(cs.options.Name, token)
		if err != nil {
			return
		}
	} else {
		value = string(token)
	}

	cookie := cs.options
	cookie.Path = "/"
	cookie.Value = value
	// Set the Expires field on the cookie based on the MaxAge
	// If MaxAge <= 0, we don't set the Expires attribute, making the cookie
	// session-only.
	if cookie.MaxAge > 0 {
		cookie.Expires = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
	}

	ctx.SetCookie(&cookie)
	return
}
