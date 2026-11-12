# secure

Security response headers, host checks and HTTPS redirects for [Iris](https://github.com/kataras/iris) v14. A port of [unrolled/secure](https://github.com/unrolled/secure).

## Install

```sh
go get github.com/iris-contrib/middleware/secure@main
```

```go
import "github.com/iris-contrib/middleware/secure"
```

## Example

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/secure"
)

func main() {
	s := secure.New(secure.Options{
		AllowedHosts:          []string{"example.com", "www.example.com"},
		SSLRedirect:           true,
		STSSeconds:            31536000,
		STSIncludeSubdomains:  true,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' $NONCE",

		// Skips the host check, the HTTPS redirect and the HSTS and HPKP
		// headers. Set it to false in production.
		IsDevelopment: true,
	})

	app := iris.New(iris.Options{
		Context: iris.ContextOptions{
			// Behind a proxy that terminates TLS, the header that tells the
			// application the original request was HTTPS.
			SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		},
	})
	app.Use(s.Handler)

	app.Get("/", func(ctx iris.Context) {
		// The nonce that replaced $NONCE in this response's policy.
		ctx.HTML("<script nonce=\"" + secure.CSPNonce(ctx) + "\">console.log('ok')</script>")
	})

	app.Listen(":8080")
}
```

## Options

Every option is off, or empty, by default.

### Host and HTTPS

| Field | Description |
| --- | --- |
| `AllowedHosts []string` | Host names the application answers for. A request for any other host gets the bad host response. |
| `AllowedHostsAreRegex bool` | Treats each `AllowedHosts` entry as a regular expression, anchored at both ends. An invalid expression panics in `New`. |
| `SSLRedirect bool` | Redirects HTTP requests to HTTPS. |
| `SSLTemporaryRedirect bool` | Redirects with 307 instead of 301. |
| `SSLHost string` | The host to redirect to. Defaults to the request's host. |
| `SSLHostFunc *SSLHostFunc` | Computes the host to redirect to from the request's host. Takes precedence over `SSLHost`. |
| `SSLForceHost bool` | Redirects to the SSL host even when the request is already HTTPS and its host differs. |
| `IsDevelopment bool` | Skips `AllowedHosts`, `SSLRedirect`, the HSTS header and the HPKP header. |

Whether a request is HTTPS is decided by `ctx.IsSSL()`, which also reads `iris.ContextOptions.SSLProxyHeaders`. The host is `ctx.Host()`, which reads `HostProxyHeaders`. Configure both when the application runs behind a proxy.

### Headers

| Field | Header |
| --- | --- |
| `STSSeconds int64` | `Strict-Transport-Security: max-age=<n>`, on HTTPS requests only. |
| `STSIncludeSubdomains bool` | Adds `; includeSubDomains`. |
| `STSPreload bool` | Adds `; preload`. |
| `ForceSTSHeader bool` | Sends the HSTS header on HTTP requests too. |
| `FrameDeny bool` | `X-Frame-Options: DENY` |
| `CustomFrameOptionsValue string` | `X-Frame-Options` with your value. Takes precedence over `FrameDeny`. |
| `ContentTypeNosniff bool` | `X-Content-Type-Options: nosniff` |
| `BrowserXSSFilter bool` | `X-XSS-Protection: 1; mode=block` |
| `CustomBrowserXSSValue string` | `X-XSS-Protection` with your value. Takes precedence over `BrowserXSSFilter`. |
| `ContentSecurityPolicy string` | `Content-Security-Policy`. `$NONCE` is replaced with `'nonce-<value>'`, a new value for each request. |
| `ContentSecurityPolicyReportOnly string` | `Content-Security-Policy-Report-Only`, with the same `$NONCE` replacement. |
| `ReferrerPolicy string` | `Referrer-Policy` |
| `FeaturePolicy string` | `Feature-Policy` |
| `PublicKey string` | `Public-Key-Pins`, on HTTPS requests only. |
| `ExpectCTHeader string` | `Expect-CT` |

Three of these headers are no longer acted on by current browsers: `Public-Key-Pins`, `Expect-CT` and `X-XSS-Protection`. `Feature-Policy` was renamed to `Permissions-Policy`, which this module does not set. The options are kept because existing configurations use them.

`SecureContextKey` is accepted for compatibility with unrolled/secure and has no effect in this port.

## Methods

| Name | Description |
| --- | --- |
| `New(options ...Options) *Secure` | Creates the middleware. Only the first `Options` value is used. |
| `(*Secure).Handler`, `(*Secure).Serve` | The handler. `Serve` calls `Handler`. |
| `(*Secure).Process(ctx) error` | Runs the checks and sets the headers from inside a handler. A non-nil error means the response was already written (bad host or redirect) and the handler should return. |
| `(*Secure).ProcessAndReturnNonce(ctx) (string, error)` | `Process`, returning the CSP nonce as well. |
| `(*Secure).ProcessNoModifyRequest(ctx) (http.Header, error)` | Runs the checks and returns the headers instead of setting them. |
| `(*Secure).SetBadHostHandler(iris.Handler)` | Replaces the bad host response, which is a 500 with the body `Bad Host`. |
| `(*Secure).WithCSPNonce(ctx, nonce)` | Sets the nonce of the current request yourself. |
| `CSPNonce(ctx) string` | The nonce of the current request, or an empty string. |

## Examples

- [_example](_example/main.go)
