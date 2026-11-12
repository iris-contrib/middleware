# throttler

Rate limiting for [Iris](https://github.com/kataras/iris) v14 through [throttled](https://github.com/throttled/throttled) v2, which implements the generic cell rate algorithm (GCRA) over a pluggable store.

Iris has a rate limiter of its own, [kataras/iris/middleware/rate](https://github.com/kataras/iris/tree/main/middleware/rate), which keeps its counters in memory. Use this module when you want throttled's stores, for example Redis, so that several instances share one limit.

## Install

```sh
go get github.com/iris-contrib/middleware/throttler@main
```

```go
import "github.com/iris-contrib/middleware/throttler"
```

## Example

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/throttler"

	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

func main() {
	app := iris.New()

	store, err := memstore.NewCtx(65536)
	if err != nil {
		panic(err)
	}

	// 20 requests per minute, with bursts of up to 5 more.
	rateLimiter, err := throttled.NewGCRARateLimiterCtx(store, throttled.RateQuota{
		MaxRate:  throttled.PerMin(20),
		MaxBurst: 5,
	})
	if err != nil {
		panic(err)
	}

	limiter := throttler.RateLimiter{
		RateLimiter: rateLimiter,
		// One quota per client address and path.
		// Without VaryBy all requests share one quota.
		VaryBy: &throttled.VaryBy{RemoteAddr: true, Path: true},
	}

	limited := app.Router("/limited", limiter.RateLimit)
	limited.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"msg": "hello"})
	})

	app.Listen(":8080")
}
```

## RateLimiter

| Field | Default | Description |
| --- | --- | --- |
| `RateLimiter` | none, required | A `throttled.RateLimiterCtx`. Decides each request and keeps the state. |
| `VaryBy` | one shared key | Anything with a `Key(*http.Request) string` method, usually a `*throttled.VaryBy`. Returns the key a request is counted under. |
| `DeniedHandler` | `DefaultDeniedHandler` | The `iris.Handler` of a request over the limit. |
| `Error` | `DefaultError` | `func(ctx iris.Context, err error)`, called when the limiter returns an error. |

`RateLimit` is the handler. It sets these response headers on every request it lets through or denies:

| Header | Value |
| --- | --- |
| `X-RateLimit-Limit` | The size of the quota. |
| `X-RateLimit-Remaining` | Requests left right now. |
| `X-RateLimit-Reset` | Seconds until the quota is full again. |
| `Retry-After` | Seconds to wait. Sent only with a denied request. |

## Default responses

| Case | Response |
| --- | --- |
| Over the limit (`DefaultDeniedHandler`) | `429 Too Many Requests`, body `limit exceeded` |
| The limiter failed, or `RateLimiter` is nil (`DefaultError`) | `500 Internal Server Error`, body `internal error` |

The limiter's error can name the address of the store, so `DefaultError` does not send it to the client. It is stored on the Context as a private error, where `ctx.GetErr()` returns it for your logs.

## Client address

`throttled.VaryBy{RemoteAddr: true}` reads `Request.RemoteAddr`, the address of the TCP peer. Behind a reverse proxy that is the proxy, and every client shares one quota. In that setup key on the header your proxy sets instead, with `throttled.VaryBy{Headers: []string{"X-Real-Ip"}}` or a `VaryBy` of your own.
