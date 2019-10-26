// Package tollboothic v2(latest) provides rate-limiting logic to iris request handlers.
package tollboothic

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/kataras/iris/v12/context"
)

// LimitHandler is a middleware that performs
// rate-limiting given a "limiter" configuration.
//
// Read more at: https://github.com/didip/tollbooth
// And https://github.com/didip/tollbooth_iris
func LimitHandler(l *limiter.Limiter) context.Handler {
	return func(ctx context.Context) {
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
