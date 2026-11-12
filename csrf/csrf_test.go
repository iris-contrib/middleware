package csrf_test

import (
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/iris-contrib/middleware/csrf"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

var authKey = []byte("9AB0F421E53A477C084477AEA06096F5")

// newApp returns an application with a GET route that answers the masked token
// and a POST route that the middleware protects.
func newApp(protect iris.Handler) *iris.Application {
	app := iris.New()
	app.Use(protect)

	app.Get("/form", func(ctx iris.Context) {
		ctx.WriteString(csrf.Token(ctx))
	})
	app.Post("/form", func(ctx iris.Context) {
		ctx.WriteString("submitted")
	})

	return app
}

// issue asks for a token and returns it with the cookie the client must send back.
func issue(e *httptest.Expect, cookieName string) (token, cookie string) {
	r := e.GET("/form").Expect().Status(httptest.StatusOK)
	return r.Body().NotEmpty().Raw(), r.Cookie(cookieName).Value().NotEmpty().Raw()
}

func TestProtect(t *testing.T) {
	app := newApp(csrf.Protect(authKey, csrf.Secure(false)))
	e := httptest.New(t, app)

	r := e.GET("/form").Expect().Status(httptest.StatusOK)
	r.Header("Vary").IsEqual("Cookie")
	token := r.Body().NotEmpty().Raw()
	cookie := r.Cookie(csrf.DefaultCookieName).Value().NotEmpty().Raw()

	// No cookie and no token.
	e.POST("/form").Expect().Status(httptest.StatusForbidden).
		Body().IsEqual(csrf.ErrBadToken.Error())

	// The cookie without a token.
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		Expect().Status(httptest.StatusForbidden).Body().IsEqual(csrf.ErrBadToken.Error())

	// A token without the cookie it belongs to.
	e.POST("/form").WithHeader(csrf.DefaultRequestHeader, token).
		Expect().Status(httptest.StatusForbidden).Body().IsEqual(csrf.ErrBadToken.Error())

	// A token that does not decode.
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithHeader(csrf.DefaultRequestHeader, "not-a-token").
		Expect().Status(httptest.StatusForbidden).Body().IsEqual(csrf.ErrBadToken.Error())

	// The token in the request header.
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithHeader(csrf.DefaultRequestHeader, token).
		Expect().Status(httptest.StatusOK).Body().IsEqual("submitted")

	// The token in a form field.
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithFormField(csrf.DefaultFieldName, token).
		Expect().Status(httptest.StatusOK).Body().IsEqual("submitted")

	// The token in a multipart form field.
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithMultipart().WithFormField(csrf.DefaultFieldName, token).
		Expect().Status(httptest.StatusOK).Body().IsEqual("submitted")
}

// The token is masked per request: two requests of one session get two
// different tokens and both are accepted.
func TestTokenIsMaskedPerRequest(t *testing.T) {
	app := newApp(csrf.Protect(authKey, csrf.Secure(false)))
	e := httptest.New(t, app)

	first, cookie := issue(e, csrf.DefaultCookieName)

	r := e.GET("/form").WithCookie(csrf.DefaultCookieName, cookie).Expect().Status(httptest.StatusOK)
	second := r.Body().NotEmpty().Raw()
	// The session already has a token, so no new cookie is issued.
	r.Cookies().IsEmpty()

	if first == second {
		t.Fatal("expected a different masked token on each request")
	}

	for _, token := range []string{first, second} {
		e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
			WithHeader(csrf.DefaultRequestHeader, token).
			Expect().Status(httptest.StatusOK)
	}
}

// The cookie the middleware issues is not readable from scripts, is sent over
// HTTPS only, is not sent on cross-site POSTs and lasts twelve hours.
func TestCookieDefaults(t *testing.T) {
	app := newApp(csrf.Protect(authKey))
	e := httptest.New(t, app)

	c := e.GET("/form").Expect().Status(httptest.StatusOK).Cookie(csrf.DefaultCookieName).Raw()
	if !c.HttpOnly {
		t.Error("expected the cookie to be HttpOnly")
	}

	if !c.Secure {
		t.Error("expected the cookie to be Secure")
	}

	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", c.SameSite)
	}

	if c.MaxAge != csrf.DefaultMaxAge {
		t.Errorf("expected Max-Age=%d, got %d", csrf.DefaultMaxAge, c.MaxAge)
	}
}

func TestCookieOptions(t *testing.T) {
	app := newApp(csrf.Protect(authKey,
		csrf.CookieName("my_csrf"),
		csrf.Secure(false),
		csrf.HTTPOnly(false),
		csrf.SameSite(http.SameSiteStrictMode),
		csrf.MaxAge(60),
		csrf.Path("/form"),
		csrf.Domain("example.com"),
	))
	e := httptest.New(t, app)

	c := e.GET("/form").Expect().Status(httptest.StatusOK).Cookie("my_csrf").Raw()
	if c.HttpOnly || c.Secure {
		t.Errorf("expected neither HttpOnly nor Secure, got HttpOnly=%v Secure=%v", c.HttpOnly, c.Secure)
	}

	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSite=Strict, got %v", c.SameSite)
	}

	if c.MaxAge != 60 || c.Path != "/form" || c.Domain != "example.com" {
		t.Errorf("expected Max-Age=60 Path=/form Domain=example.com, got Max-Age=%d Path=%s Domain=%s",
			c.MaxAge, c.Path, c.Domain)
	}
}

func TestOptions(t *testing.T) {
	var reason error

	protect := csrf.New(csrf.Options{
		Store:         csrf.NewCookieStore(authKey, csrf.Secure(false)),
		RequestHeader: "X-My-Token",
		FieldName:     "my_token",
		ErrorHandler: func(ctx iris.Context) {
			reason = csrf.FailureReason(ctx)
			ctx.StopWithText(iris.StatusTeapot, "custom")
		},
	}).Protect

	app := newApp(protect)
	app.Get("/field", func(ctx iris.Context) {
		ctx.WriteString(string(csrf.TemplateField(ctx)))
	})

	e := httptest.New(t, app)
	token, cookie := issue(e, csrf.DefaultCookieName)

	// The default header and field names are not looked at.
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithHeader(csrf.DefaultRequestHeader, token).
		Expect().Status(iris.StatusTeapot).Body().IsEqual("custom")
	if !errors.Is(reason, csrf.ErrBadToken) {
		t.Fatalf("expected the failure reason %v, got %v", csrf.ErrBadToken, reason)
	}

	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithHeader("X-My-Token", token).
		Expect().Status(httptest.StatusOK)
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithFormField("my_token", token).
		Expect().Status(httptest.StatusOK)

	field := e.GET("/field").WithCookie(csrf.DefaultCookieName, cookie).
		Expect().Status(httptest.StatusOK).Body().Raw()
	if !strings.HasPrefix(field, `<input type="hidden" name="my_token" value="`) || !strings.HasSuffix(field, `">`) {
		t.Fatalf("unexpected template field: %s", field)
	}
}

func TestNewRequiresStore(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected New to panic without a Store")
		}
	}()

	csrf.New(csrf.Options{})
}

// Filter reports the decision and leaves the response to the caller.
func TestFilter(t *testing.T) {
	c := csrf.New(csrf.Options{
		Store: csrf.NewCookieStore(authKey, csrf.Secure(false)),
		ErrorHandler: func(ctx iris.Context) {
			t.Error("the ErrorHandler must not be called by Filter")
		},
	})

	app := iris.New()
	app.Get("/form", func(ctx iris.Context) {
		if !c.Filter(ctx) {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}
		ctx.WriteString(csrf.Token(ctx))
	})
	app.Post("/form", iris.NewConditionalHandler(c.Filter, func(ctx iris.Context) {
		ctx.Values().Set("checked", true)
		ctx.Next()
	}), func(ctx iris.Context) {
		if ctx.Values().GetBool("checked") {
			ctx.WriteString("checked")
			return
		}
		ctx.WriteString("not checked")
	})

	e := httptest.New(t, app)
	token, cookie := issue(e, csrf.DefaultCookieName)

	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		WithHeader(csrf.DefaultRequestHeader, token).
		Expect().Status(httptest.StatusOK).Body().IsEqual("checked")
	// The conditional handler is skipped when the filter reports false.
	e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
		Expect().Status(httptest.StatusOK).Body().IsEqual("not checked")
}

func TestUnsafeSkipCheck(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		if ctx.Header("Authorization") != "" {
			csrf.UnsafeSkipCheck(ctx)
		}
		ctx.Next()
	})
	app.Use(csrf.Protect(authKey, csrf.Secure(false)))
	app.Post("/form", func(ctx iris.Context) { ctx.WriteString("submitted") })

	e := httptest.New(t, app)
	e.POST("/form").Expect().Status(httptest.StatusForbidden)
	e.POST("/form").WithHeader("Authorization", "Bearer api-client").
		Expect().Status(httptest.StatusOK).Body().IsEqual("submitted")
}

// On HTTPS a state-changing request must carry a Referer of the same host
// or of a trusted origin.
func TestRefererOnHTTPS(t *testing.T) {
	protect := csrf.New(csrf.Options{
		Store:          csrf.NewCookieStore(authKey),
		TrustedOrigins: []string{"ui.example.com"},
	}).Protect

	app := iris.New()
	// The test client talks to the application in memory, without TLS.
	// Give each request what the standard library gives one that arrived
	// over TLS for app.example.com.
	app.UseRouter(func(ctx iris.Context) {
		ctx.Request().TLS = &tls.ConnectionState{}
		ctx.Request().Host = "app.example.com"
		ctx.Next()
	})
	app.Use(protect)
	app.Get("/form", func(ctx iris.Context) { ctx.WriteString(csrf.Token(ctx)) })
	app.Post("/form", func(ctx iris.Context) { ctx.WriteString("submitted") })

	e := httptest.New(t, app)
	token, cookie := issue(e, csrf.DefaultCookieName)

	post := func() *httptest.Request {
		return e.POST("/form").WithCookie(csrf.DefaultCookieName, cookie).
			WithHeader(csrf.DefaultRequestHeader, token)
	}

	post().Expect().Status(httptest.StatusForbidden).Body().IsEqual(csrf.ErrNoReferer.Error())
	post().WithHeader("Referer", "https://evil.example.org/page").
		Expect().Status(httptest.StatusForbidden).Body().IsEqual(csrf.ErrBadReferer.Error())
	post().WithHeader("Referer", "https://app.example.com/form").
		Expect().Status(httptest.StatusOK).Body().IsEqual("submitted")
	post().WithHeader("Referer", "https://ui.example.com/").
		Expect().Status(httptest.StatusOK).Body().IsEqual("submitted")
}

// failingStore is a Store that has no token and cannot save one.
type failingStore struct{ err error }

func (s failingStore) Get(iris.Context) ([]byte, error) { return nil, http.ErrNoCookie }
func (s failingStore) Save(iris.Context, []byte) error  { return s.err }

// A store failure is the server's: the client gets the status alone,
// the application still reads the reason.
func TestStoreFailureIsPrivate(t *testing.T) {
	errStore := errors.New("redis: dial tcp 10.0.0.7:6379: connection refused")

	var reason error

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.Next()
		reason = csrf.FailureReason(ctx)
	})
	app.Use(csrf.New(csrf.Options{Store: failingStore{errStore}}).Protect)
	app.Get("/form", func(ctx iris.Context) { ctx.WriteString("unreachable") })

	e := httptest.New(t, app)
	e.GET("/form").Expect().Status(httptest.StatusForbidden).Body().IsEmpty()

	if !errors.Is(reason, errStore) {
		t.Fatalf("expected FailureReason to return the store error, got %v", reason)
	}
}

// A custom Store keeps the token wherever it likes; here, in memory by a session id.
type memoryStore struct{ tokens map[string][]byte }

func (s *memoryStore) Get(ctx iris.Context) ([]byte, error) {
	token, ok := s.tokens[ctx.Header("X-Session")]
	if !ok {
		return nil, errors.New("no token for this session")
	}
	return token, nil
}

func (s *memoryStore) Save(ctx iris.Context, token []byte) error {
	s.tokens[ctx.Header("X-Session")] = token
	return nil
}

func TestCustomStore(t *testing.T) {
	store := &memoryStore{tokens: make(map[string][]byte)}
	app := newApp(csrf.New(csrf.Options{Store: store}).Protect)
	e := httptest.New(t, app)

	token := e.GET("/form").WithHeader("X-Session", "one").
		Expect().Status(httptest.StatusOK).Body().NotEmpty().Raw()

	e.POST("/form").WithHeader("X-Session", "one").WithHeader(csrf.DefaultRequestHeader, token).
		Expect().Status(httptest.StatusOK).Body().IsEqual("submitted")
	// The token of one session is not valid for another.
	e.POST("/form").WithHeader("X-Session", "two").WithHeader(csrf.DefaultRequestHeader, token).
		Expect().Status(httptest.StatusForbidden).Body().IsEqual(csrf.ErrBadToken.Error())
}

func TestTokenWithoutMiddleware(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("%q %q", csrf.Token(ctx), csrf.TemplateField(ctx))
	})

	httptest.New(t, app).GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual(`"" ""`)
}
