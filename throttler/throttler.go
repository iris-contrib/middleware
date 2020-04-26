package throttler

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/kataras/iris/v12/context"

	"github.com/throttled/throttled"
)

var (
	// DefaultDeniedHandler is the default DeniedHandler for an
	// RateLimiter. It returns a 429 status code with a generic
	// message.
	DefaultDeniedHandler = func(ctx context.Context) {
		ctx.StopWithText(http.StatusTooManyRequests, "limit exceeded")
	}

	// DefaultError is the default Error function for an RateLimiter.
	// It returns a 500 status code with a generic message.
	DefaultError = func(ctx context.Context, err error) {
		ctx.StopWithError(http.StatusInternalServerError, err)
	}
)

// RateLimiter faciliates using a Limiter to limit HTTP requests.
type RateLimiter struct {
	// DeniedHandler is called if the request is disallowed. If it is
	// nil, the DefaultDeniedHandler variable is used.
	DeniedHandler context.Handler

	// Error is called if the RateLimiter returns an error. If it is
	// nil, the DefaultErrorFunc is used.
	Error func(ctx context.Context, err error)

	// Limiter is call for each request to determine whether the
	// request is permitted and update internal state. It must be set.
	RateLimiter throttled.RateLimiter

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
func (t *RateLimiter) RateLimit(ctx context.Context) {
	if t.RateLimiter == nil {
		t.error(ctx, errors.New("You must set a RateLimiter on RateLimiter"))
	}

	var k string
	if t.VaryBy != nil {
		k = t.VaryBy.Key(ctx.Request())
	}

	limited, context, err := t.RateLimiter.RateLimit(k, 1)
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

func (t *RateLimiter) error(ctx context.Context, err error) {
	e := t.Error
	if e == nil {
		e = DefaultError
	}
	e(ctx, err)
}

func setRateLimitHeaders(ctx context.Context, context throttled.RateLimitResult) {
	if v := context.Limit; v >= 0 {
		ctx.Header("X-RateLimit-Limit", strconv.Itoa(v))
	}

	if v := context.Remaining; v >= 0 {
		ctx.Header("X-RateLimit-Remaining", strconv.Itoa(v))
	}

	if v := context.ResetAfter; v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		ctx.Header("X-RateLimit-Reset", strconv.Itoa(vi))
	}

	if v := context.RetryAfter; v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		ctx.Header("Retry-After", strconv.Itoa(vi))
	}
}
