package main

import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/throttler"

	"github.com/throttled/throttled"
	"github.com/throttled/throttled/store/memstore"
)

func main() {

	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Get("/", indexHandler)

	middleware := newRateLimiterMiddleware(20)
	limited := app.Party("/limited", middleware)
	{
		limited.Get("/", limitedHandler)
	}

	// GET: http://localhost:8080
	// GET: http://localhost:8080/limited
	app.Listen(":8080")
}

// The following example demonstrates the usage of `throttler.RateLimiter` for rate-limiting access
// to 20 requests per path per minute with bursts of up to 5 additional requests.
//
// For more details and customization please read the documentation of:
// https://github.com/throttled/throttled
//
// The iris-contrib/throttler middleare is the equivalent of:
// https://github.com/throttled/throttled/blob/master/http.go.
func newRateLimiterMiddleware(maxRatePerMinute int) iris.Handler {
	store, err := memstore.New(65536)
	if err != nil {
		panic(err)
	}

	quota := throttled.RateQuota{
		MaxRate:  throttled.PerMin(maxRatePerMinute),
		MaxBurst: 5,
	}

	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		panic(err)
	}

	httpRateLimiter := throttler.RateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      &throttled.VaryBy{Path: true},
	}

	return httpRateLimiter.RateLimit
}

func indexHandler(ctx iris.Context) {
	ctx.HTML("<h1> Index Page (no limits) </h1>")
}

func limitedHandler(ctx iris.Context) {
	ctx.JSON(iris.Map{"msg": "hello"})
}
