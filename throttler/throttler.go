package throttler

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"

	"github.com/throttled/throttled/v2"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/throttler.*", "iris-contrib.throttler")
}

var (
	// DefaultDeniedHandler is the default DeniedHandler for an
	// RateLimiter. It returns a 429 status code with a generic
	// message.
	DefaultDeniedHandler = func(ctx iris.Context) {
		ctx.StopWithText(http.StatusTooManyRequests, "limit exceeded")
	}

	// DefaultError is the default Error function for an RateLimiter.
	// It returns a 500 status code with a generic message.
	//
	// The error itself is the server's (a store that cannot be reached, a
	// limiter that is not set). It is stored on the Context as a private
	// error: ctx.GetErr returns it for logging and nothing sends it to the client.
	DefaultError = func(ctx iris.Context, err error) {
		ctx.SetErr(iris.PrivateError(err))
		ctx.StopWithText(http.StatusInternalServerError, "internal error")
	}
)

// RateLimiter faciliates using a Limiter to limit HTTP requests.
type RateLimiter struct {
	// DeniedHandler is called if the request is disallowed. If it is
	// nil, the DefaultDeniedHandler variable is used.
	DeniedHandler iris.Handler

	// Error is called if the RateLimiter returns an error. If it is
	// nil, the DefaultError variable is used.
	Error func(ctx iris.Context, err error)

	// RateLimiter is called for each request to determine whether the
	// request is permitted and update internal state. It must be set.
	RateLimiter throttled.RateLimiterCtx

	// VaryBy is called for each request to generate a key for the
	// limiter. If it is nil, all requests use an empty string key.
	VaryBy interface {
		Key(*http.Request) string
	}
}

// RateLimit is an Iris middleware that limits incoming requests.
// Requests that are not limited will be passed to the handler
// unchanged.  Limited requests will be passed to the DeniedHandler.
// X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset and
// Retry-After headers will be written to the response based on the
// values in the RateLimitResult.
func (t *RateLimiter) RateLimit(ctx iris.Context) {
	if t.RateLimiter == nil {
		t.error(ctx, errors.New("you must set a RateLimiter on RateLimiter"))
		return
	}

	var k string
	if t.VaryBy != nil {
		k = t.VaryBy.Key(ctx.Request())
	}

	limited, context, err := t.RateLimiter.RateLimitCtx(ctx, k, 1)
	if err != nil {
		t.error(ctx, err)
		return
	}

	setRateLimitHeaders(ctx, context)

	if limited {
		dh := t.DeniedHandler
		if dh == nil {
			dh = DefaultDeniedHandler
		}
		dh(ctx)
		return
	}

	ctx.Next()
}

func (t *RateLimiter) error(ctx iris.Context, err error) {
	e := t.Error
	if e == nil {
		e = DefaultError
	}
	e(ctx, err)
}

func setRateLimitHeaders(ctx iris.Context, context throttled.RateLimitResult) {
	if v := context.Limit; v >= 0 {
		ctx.SetHeader("X-RateLimit-Limit", strconv.Itoa(v))
	}

	if v := context.Remaining; v >= 0 {
		ctx.SetHeader("X-RateLimit-Remaining", strconv.Itoa(v))
	}

	if v := context.ResetAfter; v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		ctx.SetHeader("X-RateLimit-Reset", strconv.Itoa(vi))
	}

	if v := context.RetryAfter; v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		ctx.SetHeader("Retry-After", strconv.Itoa(vi))
	}
}
