# tollboothic

Rate limiting for [Iris](https://github.com/kataras/iris) v14 through [tollbooth](https://github.com/didip/tollbooth) v6.

Iris has a rate limiter of its own, [kataras/iris/middleware/rate](https://github.com/kataras/iris/tree/main/middleware/rate). Use this module when you already configure tollbooth limiters, or want its limits by header, method or basic auth user.

## Install

```sh
go get github.com/iris-contrib/middleware/tollboothic@main
```

```go
import "github.com/iris-contrib/middleware/tollboothic"
```

## Example

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/iris-contrib/middleware/tollboothic"
)

func main() {
	app := iris.New()

	// One request per second for each client address. A bucket that has not
	// been used for an hour is dropped.
	l := tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
	// Behind a reverse proxy, read the client address from its header first.
	l.SetIPLookups([]string{"X-Forwarded-For", "RemoteAddr"})

	app.Get("/", tollboothic.LimitHandler(l), func(ctx iris.Context) {
		ctx.HTML("<b>Hello, world!</b>")
	})

	app.Listen(":8080")
}
```

## API

`LimitHandler(l *limiter.Limiter) iris.Handler` is the whole API. Everything else is configured on the tollbooth limiter:

| Limiter method | Default | Effect on the response to a limited request |
| --- | --- | --- |
| `SetStatusCode(int)` | 429 | The status code. |
| `SetMessage(string)` | `You have reached maximum request limit.` | The body. |
| `SetMessageContentType(string)` | `text/plain; charset=utf-8` | The `Content-Type` header. |

A limited request stops there and the route's handler does not run. Every request that goes through the handler gets tollbooth's `X-Rate-Limit-Limit`, `X-Rate-Limit-Duration` and `X-Rate-Limit-Request-Remote-Addr` headers.

See the [tollbooth documentation](https://github.com/didip/tollbooth#features) for limits by method, header and basic auth user.

## Limits

- The module is pinned to tollbooth v6. Newer major versions of tollbooth exist and have a different import path.
- tollbooth keys its buckets on the client address. A request for which it finds no address is not limited at all.

## Examples

- [_examples/limit-handler](_examples/limit-handler/main.go)
