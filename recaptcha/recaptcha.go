package recaptcha

import (
	"time"
	"fmt"
	"net/url"
	"io/ioutil"
	"encoding/json"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/netutil"
	"github.com/kataras/iris"
)

// TokenExtractor is a function that takes a context as input and returns
// either a token or an error.  An error should only be returned if an attempt
// to specify a token was found, but the information was somehow incorrectly
// formed.In the case where a token is simply not present, this should not
// be treated as an error.  An empty string should be returned in that case.
type TokenExtractor func(context.Context) (string, error)

// A function called whenever an error is encountered
type errorHandler func(context.Context, string)

const (
	responseKey = "g-recaptcha-response"
	apiURL      = "https://www.google.com/recaptcha/api/siteverify"
)

// Response is the google's recaptcha response as JSON.
type Response struct {
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Success     bool      `json:"success"`
}

// Middleware is the middleware for recaptcha implementation
type Middleware struct {
	Secret string
	Config Config
}

// Client is the default `net/http#Client` instance which
// is used to send requests to the Google API.
//
// Change Client only if you know what you're doing.
var Client = netutil.Client(time.Duration(20 * time.Second))

// OnError default error handler
func OnError(ctx context.Context, err string) {
	ctx.StatusCode(iris.StatusUnauthorized)
	ctx.Writef(err)
}

// New accepts the google's recaptcha secret key and returns
// a middleware that verifies the request by sending a response to the google's API(V2-latest).
// Secret key can be obtained by https://www.google.com/recaptcha.
//
// Used for communication between your site and Google. Be sure to keep it a secret.
//
// Use `SiteVerify` to verify a request inside another handler if needed.
func New(secret string, cfg ...Config) *Middleware {

	var c Config
	if len(cfg) == 0 {
		c = Config{}
	} else {
		c = cfg[0]
	}

	if c.Extractor == nil {
		c.Extractor = FromHeader
	}

	if c.ErrorHandler == nil {
		c.ErrorHandler = OnError
	}

	return &Middleware{Secret: secret, Config: c}
}

// FromHeader is a "TokenExtractor" that takes a give context and extracts
// the recaptcha token from the header.
func FromHeader(ctx context.Context) (string, error) {

	captchaHeader := ctx.GetHeader(responseKey)
	if captchaHeader == "" {
		return "", nil // No error, just no token
	}
	return captchaHeader, nil
}

// SiteVerify accepts context and the secret key(https://www.google.com/recaptcha)
// and returns the google's recaptcha response, if `response.Success` is true
// then validation passed.
//
// Use `New` for middleware use instead.
func (m *Middleware) SiteVerify(ctx context.Context) error {
	var response Response

	generatedResponseID, err := FromHeader(ctx)
	if err != nil || generatedResponseID == "" {
		m.Config.ErrorHandler(ctx, "captcha response not found")
		return fmt.Errorf("generated response is not valid")
	}

	if m.Secret == "" {
		m.Config.ErrorHandler(ctx, "no secret is given")
		return fmt.Errorf("secret is not valid")
	}
	r, err := Client.PostForm(apiURL,
		url.Values{
			"secret":   {m.Secret},
			"response": {generatedResponseID},
		},
	)

	if err != nil {
		m.Config.ErrorHandler(ctx, err.Error())
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return fmt.Errorf(err.Error())
	}

	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		m.Config.ErrorHandler(ctx, err.Error())
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return fmt.Errorf(err.Error())
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		m.Config.ErrorHandler(ctx, err.Error())
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return fmt.Errorf(err.Error())
	}

	if !response.Success {
		m.Config.ErrorHandler(ctx, "google verification response failed")
		return fmt.Errorf("Google verification response failed")
	}
	return nil
}

// Serve the middleware's action
func (m *Middleware) Serve(ctx context.Context) {
	if err := m.SiteVerify(ctx); err != nil {
		ctx.StopExecution()
		return
	}
	// If everything ok then call next.
	ctx.Next()
}
