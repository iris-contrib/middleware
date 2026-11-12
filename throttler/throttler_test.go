package throttler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/iris-contrib/middleware/throttler"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

// newRateLimiter returns a limiter backed by an in-memory store that allows
// one request per hour plus "burst" more: burst+1 requests pass, the next one
// is limited for the rest of the test.
func newRateLimiter(t *testing.T, burst int) throttled.RateLimiterCtx {
	t.Helper()

	store, err := memstore.NewCtx(1024)
	if err != nil {
		t.Fatal(err)
	}

	limiter, err := throttled.NewGCRARateLimiterCtx(store, throttled.RateQuota{
		MaxRate:  throttled.PerHour(1),
		MaxBurst: burst,
	})
	if err != nil {
		t.Fatal(err)
	}

	return limiter
}

func newApp(h iris.Handler) *iris.Application {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("index, not limited") })

	limited := app.Router("/limited", h)
	limited.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
	limited.Get("/other", func(ctx iris.Context) { ctx.WriteString("other") })

	return app
}

func TestRateLimit(t *testing.T) {
	limiter := &throttler.RateLimiter{RateLimiter: newRateLimiter(t, 2)}
	e := httptest.New(t, newApp(limiter.RateLimit))

	// The burst of two and the one request of the rate pass.
	for _, remaining := range []string{"2", "1", "0"} {
		r := e.GET("/limited").Expect().Status(httptest.StatusOK)
		r.Body().IsEqual("ok")
		r.Header("X-RateLimit-Limit").IsEqual("3")
		r.Header("X-RateLimit-Remaining").IsEqual(remaining)
		r.Header("X-RateLimit-Reset").NotEmpty()
		r.Headers().NotContainsKey("Retry-After")
	}

	// The fourth request is denied and told when to come back.
	r := e.GET("/limited").Expect().Status(httptest.StatusTooManyRequests)
	r.Body().IsEqual("limit exceeded")
	r.Header("X-RateLimit-Limit").IsEqual("3")
	r.Header("X-RateLimit-Remaining").IsEqual("0")
	r.Header("Retry-After").NotEmpty().NotEqual("0")

	// No VaryBy: every request shares one key, whatever its path.
	e.GET("/limited/other").Expect().Status(httptest.StatusTooManyRequests)

	// A route outside the limited router is not touched.
	r = e.GET("/").Expect().Status(httptest.StatusOK)
	r.Headers().NotContainsKey("X-RateLimit-Limit")
}

func TestVaryByPath(t *testing.T) {
	limiter := &throttler.RateLimiter{
		RateLimiter: newRateLimiter(t, 0),
		VaryBy:      &throttled.VaryBy{Path: true},
	}
	e := httptest.New(t, newApp(limiter.RateLimit))

	e.GET("/limited").Expect().Status(httptest.StatusOK)
	e.GET("/limited").Expect().Status(httptest.StatusTooManyRequests)
	// Another path has its own quota.
	e.GET("/limited/other").Expect().Status(httptest.StatusOK)
	e.GET("/limited/other").Expect().Status(httptest.StatusTooManyRequests)
}

func TestDeniedHandler(t *testing.T) {
	limiter := &throttler.RateLimiter{
		RateLimiter: newRateLimiter(t, 0),
		DeniedHandler: func(ctx iris.Context) {
			ctx.StopWithJSON(http.StatusTooManyRequests, iris.Map{"error": "slow down"})
		},
	}
	e := httptest.New(t, newApp(limiter.RateLimit))

	e.GET("/limited").Expect().Status(httptest.StatusOK)
	e.GET("/limited").Expect().Status(httptest.StatusTooManyRequests).
		JSON().Object().HasValue("error", "slow down")
}

// failingLimiter is a limiter whose store cannot be reached.
type failingLimiter struct{ err error }

func (l failingLimiter) RateLimitCtx(context.Context, string, int) (bool, throttled.RateLimitResult, error) {
	return false, throttled.RateLimitResult{}, l.err
}

// A limiter failure answers 500 with a generic message. The error, which may
// name the store's address, stays on the server.
func TestLimiterErrorIsPrivate(t *testing.T) {
	errStore := errors.New("dial tcp 10.0.0.7:6379: connection refused")
	limiter := &throttler.RateLimiter{RateLimiter: failingLimiter{errStore}}

	var stored error

	app := newApp(limiter.RateLimit)
	app.UseRouter(func(ctx iris.Context) {
		ctx.Next()
		stored = ctx.GetErr()
	})

	e := httptest.New(t, app)
	e.GET("/limited").Expect().Status(httptest.StatusInternalServerError).
		Body().IsEqual("internal error")

	if !errors.Is(stored, errStore) {
		t.Fatalf("expected the limiter error on the Context, got %v", stored)
	}
}

func TestErrorFunc(t *testing.T) {
	errStore := errors.New("store down")
	limiter := &throttler.RateLimiter{
		RateLimiter: failingLimiter{errStore},
		Error: func(ctx iris.Context, err error) {
			if !errors.Is(err, errStore) {
				t.Errorf("expected %v, got %v", errStore, err)
			}
			ctx.StopWithText(http.StatusServiceUnavailable, "try later")
		},
	}

	e := httptest.New(t, newApp(limiter.RateLimit))
	e.GET("/limited").Expect().Status(httptest.StatusServiceUnavailable).Body().IsEqual("try later")
}

// A RateLimiter value without a limiter is a configuration error:
// it answers 500 and does not reach the handler.
func TestMissingRateLimiter(t *testing.T) {
	limiter := new(throttler.RateLimiter)

	e := httptest.New(t, newApp(limiter.RateLimit))
	e.GET("/limited").Expect().Status(httptest.StatusInternalServerError).
		Body().IsEqual("internal error")
}
