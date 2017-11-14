package csrf

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kataras/iris/context"

	"github.com/gorilla/securecookie"
)

// store represents the session storage used for CSRF tokens.
type store interface {
	// Get returns the real CSRF token from the store.
	Get(context.Context) ([]byte, error)
	// Save stores the real CSRF token in the store and writes a
	// cookie to the Context's http.ResponseWriter.
	// For non-cookie stores, the cookie should contain a unique (256 bit) ID
	// or key that references the token in the backend store.
	// csrf.GenerateRandomBytes is a helper function for generating secure IDs.
	Save(ctx context.Context, token []byte) error
}

// cookieStore is a signed cookie session store for CSRF tokens.
type cookieStore struct {
	name     string
	maxAge   int
	secure   bool
	httpOnly bool
	path     string
	domain   string
	sc       *securecookie.SecureCookie
}

// Get retrieves a CSRF token from the session cookie. It returns an empty token
// if decoding fails (e.g. HMAC validation fails or the named cookie doesn't exist).
func (cs *cookieStore) Get(ctx context.Context) ([]byte, error) {
	// Retrieve the cookie from the request
	cookieValue := ctx.GetCookie(cs.name)
	if cookieValue == "" {
		return nil, fmt.Errorf("empty cookie")
	}

	token := make([]byte, tokenLength)
	// Decode the HMAC authenticated cookie.
	err := cs.sc.Decode(cs.name, cookieValue, &token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// Save stores the CSRF token in the session cookie.
func (cs *cookieStore) Save(ctx context.Context, token []byte) error {
	// Generate an encoded cookie value with the CSRF token.
	encoded, err := cs.sc.Encode(cs.name, token)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     cs.name,
		Value:    encoded,
		MaxAge:   cs.maxAge,
		HttpOnly: cs.httpOnly,
		Secure:   cs.secure, // if true but not https then it fails ofc, so be careful.
		Path:     cs.path,
		Domain:   cs.domain,
	}

	// Set the Expires field on the cookie based on the MaxAge
	// If MaxAge <= 0, we don't set the Expires attribute, making the cookie
	// session-only.
	if cs.maxAge > 0 {
		cookie.Expires = time.Now().Add(
			time.Duration(cs.maxAge) * time.Second)
	}

	// Write the authenticated cookie to the response.
	ctx.SetCookie(cookie)

	return nil
}
