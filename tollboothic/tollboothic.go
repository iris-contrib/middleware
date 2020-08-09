// Package tollboothic v6(latest) provides rate-limiting logic to iris request handlers.
package tollboothic

import (
	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/kataras/iris/v12"
)

// LimitHandler is a middleware that performs
// rate-limiting given a "limiter" configuration.
//
// Read more at: https://github.com/didip/tollbooth.
func LimitHandler(l *limiter.Limiter) iris.Handler {
	return func(ctx iris.Context) {
		httpError := tollbooth.LimitByRequest(l, ctx.ResponseWriter(), ctx.Request())
		if httpError != nil {
			ctx.ContentType(l.GetMessageContentType())
			ctx.StatusCode(httpError.StatusCode)
			ctx.WriteString(httpError.Message)
			ctx.StopExecution()
			return
		}

		ctx.Next()
	}
}
