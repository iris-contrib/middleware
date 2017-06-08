// This has been cloned to work with Iris,
// credits goes to https://github.com/unrolled/secure , I did nothing special here.

package secure

import (
	"fmt"
	"net"
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

const (
	stsHeader           = "Strict-Transport-Security"
	stsSubdomainString  = "; includeSubdomains"
	stsPreloadString    = "; preload"
	frameOptionsHeader  = "X-Frame-Options"
	frameOptionsValue   = "DENY"
	contentTypeHeader   = "X-Content-Type-Options"
	contentTypeValue    = "nosniff"
	xssProtectionHeader = "X-XSS-Protection"
	xssProtectionValue  = "1; mode=block"
	cspHeader           = "Content-Security-Policy"
	hpkpHeader          = "Public-Key-Pins"
)

func defaultBadHostHandler(ctx context.Context) {
	ctx.StatusCode(iris.StatusInternalServerError)
	ctx.Text("Bad Host")
}

// Options is a struct for specifying configuration options for the secure.Secure middleware.
type Options struct {
	// AllowedHosts is a list of fully qualified domain names that are allowed. Default is empty list, which allows any and all host names.
	AllowedHosts []string
	// If SSLRedirect is set to true, then only allow https requests. Default is false.
	SSLRedirect bool
	// If SSLTemporaryRedirect is true, the a 302 will be used while redirecting. Default is false (301).
	SSLTemporaryRedirect bool
	// SSLHost is the host name that is used to redirect http requests to https. Default is "", which indicates to use the same host.
	SSLHost string
	// SSLProxyHeaders is set of header keys with associated values that would indicate a valid https request. Useful when using Nginx: `map[string]string{"X-Forwarded-Proto": "https"}`. Default is blank map.
	SSLProxyHeaders map[string]string
	// STSSeconds is the max-age of the Strict-Transport-Security header. Default is 0, which would NOT include the header.
	STSSeconds int64
	// If STSIncludeSubdomains is set to true, the `includeSubdomains` will be appended to the Strict-Transport-Security header. Default is false.
	STSIncludeSubdomains bool
	// If STSPreload is set to true, the `preload` flag will be appended to the Strict-Transport-Security header. Default is false.
	STSPreload bool
	// If ForceSTSHeader is set to true, the STS header will be added even when the connection is HTTP. Default is false.
	ForceSTSHeader bool
	// If FrameDeny is set to true, adds the X-Frame-Options header with the value of `DENY`. Default is false.
	FrameDeny bool
	// CustomFrameOptionsValue allows the X-Frame-Options header value to be set with a custom value. This overrides the FrameDeny option.
	CustomFrameOptionsValue string
	// If ContentTypeNosniff is true, adds the X-Content-Type-Options header with the value `nosniff`. Default is false.
	ContentTypeNosniff bool
	// BrowserXSSFilter If it's true, adds the X-XSS-Protection header with the value `1; mode=block`. Default is false.
	BrowserXSSFilter bool
	// ContentSecurityPolicy allows the Content-Security-Policy header value to be set with a custom value. Default is "".
	ContentSecurityPolicy string
	// PublicKey implements HPKP to prevent MITM attacks with forged certificates. Default is "".
	PublicKey string
	// When developing, the AllowedHosts, SSL, and STS options can cause some unwanted effects. Usually testing happens on http, not https, and on localhost, not your production domain... so set this to true for dev environment.
	// If you would like your development environment to mimic production with complete Host blocking, SSL redirects, and STS headers, leave this as false. Default if false.
	IsDevelopment bool
	// Option to ignore private lan ips. This could be helpful to prevent internal health checks from upgrading to SSL
	IgnorePrivateIPs bool
}

// Secure is a middleware that helps setup a few basic security features. A single secure.Options struct can be
// provided to configure which features should be enabled, and the ability to override a few of the default values.
type Secure struct {
	// Customize Secure with an Options struct.
	opt Options

	// Handlers for when an error occurs (ie bad host).
	badHostHandler context.Handler
}

// New constructs a new Secure instance with supplied options.
func New(options ...Options) *Secure {
	var o Options
	if len(options) == 0 {
		o = Options{}
	} else {
		o = options[0]
	}

	return &Secure{
		opt:            o,
		badHostHandler: defaultBadHostHandler,
	}
}

// SetBadHostHandler sets the handler to call when secure rejects the host name.
func (s *Secure) SetBadHostHandler(handler context.Handler) {
	s.badHostHandler = handler
}

// Serve implements the context.HandlerFunc for integration with iris.
func (s *Secure) Serve(ctx context.Context) {
	// Let secure process the request. If it returns an error,
	// that indicates the request should not continue.
	err := s.Process(ctx)

	// If there was an error, do not continue.
	if err != nil {
		return
	}

	ctx.Next()
}

// Process runs the actual checks and returns an error if the middleware chain should stop.
func (s *Secure) Process(ctx context.Context) error {
	// We've hit an private ip and the option is enabled
	if s.opt.IgnorePrivateIPs && isPrivateSubnet(net.ParseIP(ctx.RemoteAddr())) {
		return nil
	}

	// Allowed hosts check.
	if len(s.opt.AllowedHosts) > 0 && !s.opt.IsDevelopment {
		isGoodHost := false
		for _, allowedHost := range s.opt.AllowedHosts {
			if strings.EqualFold(allowedHost, string(ctx.Host())) {
				isGoodHost = true
				break
			}
		}

		if !isGoodHost {
			s.badHostHandler(ctx)
			return fmt.Errorf("Bad host name: %s", string(ctx.Host()))
		}
	}

	// Determine if we are on HTTPS.
	isSSL := strings.EqualFold(string(ctx.Request().URL.Scheme), "https")
	if !isSSL {
		for k, v := range s.opt.SSLProxyHeaders {
			if ctx.GetHeader(k) == v {
				isSSL = true
				break
			}
		}
	}

	// SSL check.
	if s.opt.SSLRedirect && !isSSL && !s.opt.IsDevelopment {
		url := ctx.Request().URL
		url.Scheme = "https"
		url.Host = ctx.Host()

		if len(s.opt.SSLHost) > 0 {
			url.Host = s.opt.SSLHost
		}

		status := iris.StatusMovedPermanently
		if s.opt.SSLTemporaryRedirect {
			status = iris.StatusTemporaryRedirect
		}

		ctx.Redirect(url.String(), status)
		return fmt.Errorf("Redirecting to HTTPS")
	}

	// Strict Transport Security header. Only add header when we know it's an SSL connection.
	// See https://tools.ietf.org/html/rfc6797#section-7.2 for details.
	if s.opt.STSSeconds != 0 && (isSSL || s.opt.ForceSTSHeader) && !s.opt.IsDevelopment {
		stsSub := ""
		if s.opt.STSIncludeSubdomains {
			stsSub = stsSubdomainString
		}

		if s.opt.STSPreload {
			stsSub += stsPreloadString
		}
		ctx.Header(stsHeader, fmt.Sprintf("max-age=%d%s", s.opt.STSSeconds, stsSub))

	}

	// Frame Options header.
	if len(s.opt.CustomFrameOptionsValue) > 0 {
		ctx.Header(frameOptionsHeader, s.opt.CustomFrameOptionsValue)
	} else if s.opt.FrameDeny {
		ctx.Header(frameOptionsHeader, frameOptionsValue)
	}

	// Content Type Options header.
	if s.opt.ContentTypeNosniff {
		ctx.Header(contentTypeHeader, contentTypeValue)
	}

	// XSS Protection header.
	if s.opt.BrowserXSSFilter {
		ctx.Header(xssProtectionHeader, xssProtectionValue)
	}

	// HPKP header.
	if len(s.opt.PublicKey) > 0 && isSSL && !s.opt.IsDevelopment {
		ctx.Header(hpkpHeader, s.opt.PublicKey)
	}

	// Content Security Policy header.
	if len(s.opt.ContentSecurityPolicy) > 0 {
		ctx.Header(cspHeader, s.opt.ContentSecurityPolicy)
	}

	return nil
}
