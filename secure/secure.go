// This has been cloned to work with iris,
// credits goes to https://github.com/unrolled/secure , I did nothing special here.

package secure

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/kataras/iris/v12"
)

const (
	stsHeader            = "Strict-Transport-Security"
	stsSubdomainString   = "; includeSubDomains"
	stsPreloadString     = "; preload"
	frameOptionsHeader   = "X-Frame-Options"
	frameOptionsValue    = "DENY"
	contentTypeHeader    = "X-Content-Type-Options"
	contentTypeValue     = "nosniff"
	xssProtectionHeader  = "X-XSS-Protection"
	xssProtectionValue   = "1; mode=block"
	cspHeader            = "Content-Security-Policy"
	cspReportOnlyHeader  = "Content-Security-Policy-Report-Only"
	hpkpHeader           = "Public-Key-Pins"
	referrerPolicyHeader = "Referrer-Policy"
	featurePolicyHeader  = "Feature-Policy"
	expectCTHeader       = "Expect-CT"

	ctxDefaultSecureHeaderKey = "SecureResponseHeader"
	cspNonceSize              = 16
)

// SSLHostFunc a type whose pointer is the type of field `SSLHostFunc` of `Options` struct
type SSLHostFunc func(host string) (newHost string)

func defaultBadHostHandler(ctx iris.Context) {
	ctx.StopWithText(iris.StatusInternalServerError, "Bad Host")
}

// Options is a struct for specifying configuration options for the secure.Secure middleware.
type Options struct {
	// If BrowserXSSFilter is true, adds the X-XSS-Protection header with the value `1; mode=block`. Default is false.
	BrowserXSSFilter bool // nolint: golint
	// If ContentTypeNosniff is true, adds the X-Content-Type-Options header with the value `nosniff`. Default is false.
	ContentTypeNosniff bool
	// If ForceSTSHeader is set to true, the STS header will be added even when the connection is HTTP. Default is false.
	ForceSTSHeader bool
	// If FrameDeny is set to true, adds the X-Frame-Options header with the value of `DENY`. Default is false.
	FrameDeny bool
	// When developing, the AllowedHosts, SSL, and STS options can cause some unwanted effects. Usually testing happens on http, not https, and on localhost, not your production domain... so set this to true for dev environment.
	// If you would like your development environment to mimic production with complete Host blocking, SSL redirects, and STS headers, leave this as false. Default if false.
	IsDevelopment bool
	// nonceEnabled is used internally for dynamic nouces.
	nonceEnabled bool
	// If SSLRedirect is set to true, then only allow https requests. Default is false.
	SSLRedirect bool
	// If SSLForceHost is true and SSLHost is set, requests will be forced to use SSLHost even the ones that are already using SSL. Default is false.
	SSLForceHost bool
	// If SSLTemporaryRedirect is true, the a 302 will be used while redirecting. Default is false (301).
	SSLTemporaryRedirect bool
	// If STSIncludeSubdomains is set to true, the `includeSubdomains` will be appended to the Strict-Transport-Security header. Default is false.
	STSIncludeSubdomains bool
	// If STSPreload is set to true, the `preload` flag will be appended to the Strict-Transport-Security header. Default is false.
	STSPreload bool
	// ContentSecurityPolicy allows the Content-Security-Policy header value to be set with a custom value. Default is "".
	ContentSecurityPolicy string
	// ContentSecurityPolicyReportOnly allows the Content-Security-Policy-Report-Only header value to be set with a custom value. Default is "".
	ContentSecurityPolicyReportOnly string
	// CustomBrowserXSSValue allows the X-XSS-Protection header value to be set with a custom value. This overrides the BrowserXSSFilter option. Default is "".
	CustomBrowserXSSValue string // nolint: golint
	// Passing a template string will replace `$NONCE` with a dynamic nonce value of 16 bytes for each request which can be later retrieved using the Nonce function.
	// Eg: script-src $NONCE -> script-src 'nonce-a2ZobGFoZg=='
	// CustomFrameOptionsValue allows the X-Frame-Options header value to be set with a custom value. This overrides the FrameDeny option. Default is "".
	CustomFrameOptionsValue string
	// PublicKey implements HPKP to prevent MITM attacks with forged certificates. Default is "".
	PublicKey string
	// ReferrerPolicy allows sites to control when browsers will pass the Referer header to other sites. Default is "".
	ReferrerPolicy string
	// FeaturePolicy allows to selectively enable and disable use of various browser features and APIs. Default is "".
	FeaturePolicy string
	// SSLHost is the host name that is used to redirect http requests to https. Default is "", which indicates to use the same host.
	SSLHost string
	// AllowedHosts is a list of fully qualified domain names that are allowed. Default is empty list, which allows any and all host names.
	AllowedHosts []string
	// AllowedHostsAreRegex determines, if the provided slice contains valid regular expressions. If this flag is set to true, every request's
	// host will be checked against these expressions. Default is false for backwards compatibility.
	AllowedHostsAreRegex bool
	// SSLHostFunc is a function pointer, the return value of the function is the host name that has same functionality as `SSHost`. Default is nil.
	// If SSLHostFunc is nil, the `SSLHost` option will be used.
	SSLHostFunc *SSLHostFunc
	// STSSeconds is the max-age of the Strict-Transport-Security header. Default is 0, which would NOT include the header.
	STSSeconds int64
	// ExpectCTHeader allows the Expect-CT header value to be set with a custom value. Default is "".
	ExpectCTHeader string
	// SecureContextKey allows a custom key to be specified for context storage.
	SecureContextKey string
}

// Secure is a middleware that helps setup a few basic security features. A single secure.Options struct can be
// provided to configure which features should be enabled, and the ability to override a few of the default values.
type Secure struct {
	// Customize Secure with an Options struct.
	opt Options

	// badHostHandler is the handler used when an incorrect host is passed in.
	badHostHandler iris.Handler

	// cRegexAllowedHosts saves the compiled regular expressions of the AllowedHosts
	// option for subsequent use in processRequest
	cRegexAllowedHosts []*regexp.Regexp

	// ctxSecureHeaderKey is the key used for context storage for request modification.
	ctxSecureHeaderKey string
}

// New constructs a new Secure instance with the supplied options.
func New(options ...Options) *Secure {
	var o Options
	if len(options) == 0 {
		o = Options{}
	} else {
		o = options[0]
	}

	o.ContentSecurityPolicy = strings.Replace(o.ContentSecurityPolicy, "$NONCE", "'nonce-%[1]s'", -1)
	o.ContentSecurityPolicyReportOnly = strings.Replace(o.ContentSecurityPolicyReportOnly, "$NONCE", "'nonce-%[1]s'", -1)

	o.nonceEnabled = strings.Contains(o.ContentSecurityPolicy, "%[1]s") || strings.Contains(o.ContentSecurityPolicyReportOnly, "%[1]s")

	s := &Secure{
		opt:            o,
		badHostHandler: defaultBadHostHandler,
	}

	if s.opt.AllowedHostsAreRegex {
		// Test for invalid regular expressions in AllowedHosts
		for _, allowedHost := range o.AllowedHosts {
			regex, err := regexp.Compile(fmt.Sprintf("^%s$", allowedHost))
			if err != nil {
				panic(fmt.Sprintf("Error parsing AllowedHost: %s", err))
			}
			s.cRegexAllowedHosts = append(s.cRegexAllowedHosts, regex)
		}
	}

	s.ctxSecureHeaderKey = ctxDefaultSecureHeaderKey
	if len(s.opt.SecureContextKey) > 0 {
		s.ctxSecureHeaderKey = s.opt.SecureContextKey
	}

	return s
}

// SetBadHostHandler sets the handler to call when secure rejects the host name.
func (s *Secure) SetBadHostHandler(handler iris.Handler) {
	s.badHostHandler = handler
}

// Process runs the actual checks and writes the headers in the Context.
func (s *Secure) Process(ctx iris.Context) error {
	responseHeader, err := s.processRequest(ctx)
	addResponseHeaders(responseHeader, ctx)

	return err
}

// Handler is the main middleware.
func (s *Secure) Handler(ctx iris.Context) {
	// Let secure process the request. If it returns an error,
	// that indicates the request should not continue.
	responseHeader, err := s.processRequest(ctx)
	addResponseHeaders(responseHeader, ctx)

	// If there was an error, do not continue.
	if err != nil {
		ctx.StopExecution()
		return
	}

	// Avoid header rewrite if response is a redirection.
	if status := ctx.GetStatusCode(); status > 300 && status < 399 {
		ctx.StopExecution()
		return
	}

	ctx.Next()
}

// Serve same as "Handler". It's the main middleware's handler.
func (s *Secure) Serve(ctx iris.Context) {
	s.Handler(ctx)
}

// addResponseHeaders Adds the headers from 'responseHeader' to the Context.
func addResponseHeaders(responseHeader http.Header, ctx iris.Context) {
	for key, values := range responseHeader {
		for _, value := range values {
			ctx.Header(key, value)
		}
	}
}

// ProcessAndReturnNonce runs the actual checks and writes the headers in the ResponseWriter.
// In addition, the generated nonce for the request is returned as well as the error value.
func (s *Secure) ProcessAndReturnNonce(ctx iris.Context) (string, error) {
	responseHeader, err := s.processRequest(ctx)
	if err != nil {
		return "", err
	}

	addResponseHeaders(responseHeader, ctx)

	return CSPNonce(ctx), err
}

// ProcessNoModifyRequest runs the actual checks but does not write the headers in the Context.
func (s *Secure) ProcessNoModifyRequest(ctx iris.Context) (http.Header, error) {
	return s.processRequest(ctx)
}

// WithCSPNonce sets the Context's "iris.secure.nonce" value to the given nonce.
func (s *Secure) WithCSPNonce(ctx iris.Context, nonce string) {
	ctx.Values().Set(cspNonceKey, nonce)
}

// processRequest runs the actual checks on the request and returns an error if the middleware chain should stop.
func (s *Secure) processRequest(ctx iris.Context) (http.Header, error) {
	// Setup nonce if required.
	if s.opt.nonceEnabled {
		s.WithCSPNonce(ctx, cspRandNonce())
	}

	// Resolve the host for the request, using proxy headers if present.
	host := ctx.Host()

	// Allowed hosts check.
	if len(s.opt.AllowedHosts) > 0 && !s.opt.IsDevelopment {
		isGoodHost := false
		if s.opt.AllowedHostsAreRegex {
			for _, allowedHost := range s.cRegexAllowedHosts {
				if match := allowedHost.MatchString(host); match {
					isGoodHost = true
					break
				}
			}
		} else {
			for _, allowedHost := range s.opt.AllowedHosts {
				if strings.EqualFold(allowedHost, host) {
					isGoodHost = true
					break
				}
			}
		}

		if !isGoodHost {
			s.badHostHandler(ctx)
			return nil, fmt.Errorf("bad host name: %s", host)
		}
	}

	// Determine if we are on HTTPS.
	ssl := ctx.IsSSL()

	// SSL check.
	if s.opt.SSLRedirect && !ssl && !s.opt.IsDevelopment {
		r := ctx.Request()
		url := r.URL
		url.Scheme = "https"
		url.Host = host

		if s.opt.SSLHostFunc != nil {
			if h := (*s.opt.SSLHostFunc)(host); len(h) > 0 {
				url.Host = h
			}
		} else if len(s.opt.SSLHost) > 0 {
			url.Host = s.opt.SSLHost
		}

		status := iris.StatusMovedPermanently
		if s.opt.SSLTemporaryRedirect {
			status = iris.StatusTemporaryRedirect
		}

		ctx.Redirect(url.String(), status)
		return nil, fmt.Errorf("redirecting to HTTPS")
	}

	if s.opt.SSLForceHost {
		var SSLHost = host
		if s.opt.SSLHostFunc != nil {
			if h := (*s.opt.SSLHostFunc)(host); len(h) > 0 {
				SSLHost = h
			}
		} else if len(s.opt.SSLHost) > 0 {
			SSLHost = s.opt.SSLHost
		}
		if SSLHost != host {
			r := ctx.Request()
			url := r.URL
			url.Scheme = "https"
			url.Host = SSLHost

			status := iris.StatusMovedPermanently
			if s.opt.SSLTemporaryRedirect {
				status = iris.StatusTemporaryRedirect
			}

			ctx.Redirect(url.String(), status)
			return nil, fmt.Errorf("redirecting to HTTPS")
		}
	}

	// Create our header container.
	responseHeader := make(http.Header)

	// Strict Transport Security header. Only add header when we know it's an SSL connection.
	// See https://tools.ietf.org/html/rfc6797#section-7.2 for details.
	if s.opt.STSSeconds != 0 && (ssl || s.opt.ForceSTSHeader) && !s.opt.IsDevelopment {
		stsSub := ""
		if s.opt.STSIncludeSubdomains {
			stsSub = stsSubdomainString
		}

		if s.opt.STSPreload {
			stsSub += stsPreloadString
		}

		responseHeader.Set(stsHeader, fmt.Sprintf("max-age=%d%s", s.opt.STSSeconds, stsSub))
	}

	// Frame Options header.
	if len(s.opt.CustomFrameOptionsValue) > 0 {
		responseHeader.Set(frameOptionsHeader, s.opt.CustomFrameOptionsValue)
	} else if s.opt.FrameDeny {
		responseHeader.Set(frameOptionsHeader, frameOptionsValue)
	}

	// Content Type Options header.
	if s.opt.ContentTypeNosniff {
		responseHeader.Set(contentTypeHeader, contentTypeValue)
	}

	// XSS Protection header.
	if len(s.opt.CustomBrowserXSSValue) > 0 {
		responseHeader.Set(xssProtectionHeader, s.opt.CustomBrowserXSSValue)
	} else if s.opt.BrowserXSSFilter {
		responseHeader.Set(xssProtectionHeader, xssProtectionValue)
	}

	// HPKP header.
	if len(s.opt.PublicKey) > 0 && ssl && !s.opt.IsDevelopment {
		responseHeader.Set(hpkpHeader, s.opt.PublicKey)
	}

	// Content Security Policy header.
	if len(s.opt.ContentSecurityPolicy) > 0 {
		if s.opt.nonceEnabled {
			responseHeader.Set(cspHeader, fmt.Sprintf(s.opt.ContentSecurityPolicy, CSPNonce(ctx)))
		} else {
			responseHeader.Set(cspHeader, s.opt.ContentSecurityPolicy)
		}
	}

	// Content Security Policy Report Only header.
	if len(s.opt.ContentSecurityPolicyReportOnly) > 0 {
		if s.opt.nonceEnabled {
			responseHeader.Set(cspReportOnlyHeader, fmt.Sprintf(s.opt.ContentSecurityPolicyReportOnly, CSPNonce(ctx)))
		} else {
			responseHeader.Set(cspReportOnlyHeader, s.opt.ContentSecurityPolicyReportOnly)
		}
	}

	// Referrer Policy header.
	if len(s.opt.ReferrerPolicy) > 0 {
		responseHeader.Set(referrerPolicyHeader, s.opt.ReferrerPolicy)
	}

	// Feature Policy header.
	if len(s.opt.FeaturePolicy) > 0 {
		responseHeader.Set(featurePolicyHeader, s.opt.FeaturePolicy)
	}

	// Expect-CT header.
	if len(s.opt.ExpectCTHeader) > 0 {
		responseHeader.Set(expectCTHeader, s.opt.ExpectCTHeader)
	}

	return responseHeader, nil
}
