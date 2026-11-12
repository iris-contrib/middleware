package secure_test

import (
	"net/http"
	stdhttptest "net/http/httptest"
	"testing"

	"github.com/iris-contrib/middleware/secure"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func newApp(s *secure.Secure, opts ...iris.Configurator) *iris.Application {
	app := iris.New(opts...)
	app.Use(s.Handler)
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	return app
}

func TestHeaders(t *testing.T) {
	s := secure.New(secure.Options{
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXSSFilter:      true,
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "same-origin",
		FeaturePolicy:         "vibrate 'none'",
		ExpectCTHeader:        "enforce, max-age=30",
		STSSeconds:            31536000,
		STSIncludeSubdomains:  true,
		STSPreload:            true,
	})

	e := httptest.New(t, newApp(s))

	r := e.GET("/").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("ok")
	r.Header("X-Frame-Options").IsEqual("DENY")
	r.Header("X-Content-Type-Options").IsEqual("nosniff")
	r.Header("X-XSS-Protection").IsEqual("1; mode=block")
	r.Header("Content-Security-Policy").IsEqual("default-src 'self'")
	r.Header("Referrer-Policy").IsEqual("same-origin")
	r.Header("Feature-Policy").IsEqual("vibrate 'none'")
	r.Header("Expect-CT").IsEqual("enforce, max-age=30")
	// The request is plain HTTP, so no HSTS header.
	r.Headers().NotContainsKey("Strict-Transport-Security")
}

func TestCustomHeaderValues(t *testing.T) {
	s := secure.New(secure.Options{
		FrameDeny:               true,
		CustomFrameOptionsValue: "SAMEORIGIN",
		BrowserXSSFilter:        true,
		CustomBrowserXSSValue:   "0",
	})

	r := httptest.New(t, newApp(s)).GET("/").Expect().Status(httptest.StatusOK)
	r.Header("X-Frame-Options").IsEqual("SAMEORIGIN")
	r.Header("X-XSS-Protection").IsEqual("0")
}

// The HSTS header is sent on HTTPS. Behind a proxy that terminates TLS the
// framework's SSLProxyHeaders option is what marks a request as HTTPS.
func TestSTSHeader(t *testing.T) {
	opts := secure.Options{STSSeconds: 600, STSIncludeSubdomains: true}

	behindProxy := iris.Options{Context: iris.ContextOptions{
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	}}

	e := httptest.New(t, newApp(secure.New(opts), behindProxy))
	e.GET("/").Expect().Status(httptest.StatusOK).Headers().NotContainsKey("Strict-Transport-Security")
	e.GET("/").WithHeader("X-Forwarded-Proto", "https").Expect().Status(httptest.StatusOK).
		Header("Strict-Transport-Security").IsEqual("max-age=600; includeSubDomains")

	opts.ForceSTSHeader = true
	e = httptest.New(t, newApp(secure.New(opts)))
	e.GET("/").Expect().Status(httptest.StatusOK).
		Header("Strict-Transport-Security").IsEqual("max-age=600; includeSubDomains")
}

func TestAllowedHosts(t *testing.T) {
	s := secure.New(secure.Options{AllowedHosts: []string{"example.com"}})
	e := httptest.New(t, newApp(s))

	e.GET("/").WithHost("example.com").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
	e.GET("/").WithHost("EXAMPLE.com").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHost("evil.example.org").Expect().Status(httptest.StatusInternalServerError).
		Body().IsEqual("Bad Host")

	s = secure.New(secure.Options{
		AllowedHosts:         []string{`.*\.example\.com`},
		AllowedHostsAreRegex: true,
	})
	s.SetBadHostHandler(func(ctx iris.Context) {
		ctx.StopWithText(iris.StatusMisdirectedRequest, "wrong host")
	})
	e = httptest.New(t, newApp(s))

	e.GET("/").WithHost("api.example.com").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHost("example.org").Expect().Status(iris.StatusMisdirectedRequest).
		Body().IsEqual("wrong host")
}

// redirect sends one request and returns the status and the Location of the
// answer, without following it.
func redirect(t *testing.T, app *iris.Application, target string, header http.Header) (int, string) {
	t.Helper()

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	req := stdhttptest.NewRequest(http.MethodGet, target, nil)
	for name, values := range header {
		req.Header[name] = values
	}

	rec := stdhttptest.NewRecorder()
	app.ServeHTTP(rec, req)

	return rec.Code, rec.Header().Get("Location")
}

func TestSSLRedirect(t *testing.T) {
	behindProxy := iris.Options{Context: iris.ContextOptions{
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	}}

	app := newApp(secure.New(secure.Options{SSLRedirect: true}), behindProxy)

	status, location := redirect(t, app, "http://example.com/?a=b", nil)
	if status != http.StatusMovedPermanently || location != "https://example.com/?a=b" {
		t.Fatalf("expected a 301 to https://example.com/?a=b, got %d to %q", status, location)
	}

	// Already HTTPS: served.
	status, location = redirect(t, app, "http://example.com/", http.Header{"X-Forwarded-Proto": []string{"https"}})
	if status != http.StatusOK || location != "" {
		t.Fatalf("expected an HTTPS request to be served, got %d to %q", status, location)
	}

	app = newApp(secure.New(secure.Options{
		SSLRedirect:          true,
		SSLTemporaryRedirect: true,
		SSLHost:              "secure.example.com",
	}))

	status, location = redirect(t, app, "http://example.com/", nil)
	if status != http.StatusTemporaryRedirect || location != "https://secure.example.com/" {
		t.Fatalf("expected a 307 to https://secure.example.com/, got %d to %q", status, location)
	}
}

// IsDevelopment switches the host check, the redirect and HSTS off
// and leaves the other headers on.
func TestIsDevelopment(t *testing.T) {
	s := secure.New(secure.Options{
		AllowedHosts:   []string{"example.com"},
		SSLRedirect:    true,
		STSSeconds:     600,
		ForceSTSHeader: true,
		FrameDeny:      true,
		IsDevelopment:  true,
	})

	r := httptest.New(t, newApp(s)).GET("/").WithHost("localhost").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("ok")
	r.Header("X-Frame-Options").IsEqual("DENY")
	r.Headers().NotContainsKey("Strict-Transport-Security")
}

// Process runs the checks from inside a handler.
func TestProcess(t *testing.T) {
	s := secure.New(secure.Options{
		AllowedHosts:       []string{"example.com"},
		ContentTypeNosniff: true,
	})

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		if err := s.Process(ctx); err != nil {
			return // the bad host response is already written.
		}
		ctx.WriteString("processed")
	})

	e := httptest.New(t, app)
	r := e.GET("/").WithHost("example.com").Expect().Status(httptest.StatusOK)
	r.Body().IsEqual("processed")
	r.Header("X-Content-Type-Options").IsEqual("nosniff")

	e.GET("/").WithHost("example.org").Expect().Status(httptest.StatusInternalServerError).
		Body().IsEqual("Bad Host")
}
