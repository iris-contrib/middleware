// Package tollboothic provides rate-limiting logic to iris request handlers.
package tollboothic

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/config"
	"github.com/kataras/iris/context"
)

// LimitHandler is a middleware that performs
// rate-limiting given a "limiter" configuration.
//
// Read more at: https://github.com/didip/tollbooth
func LimitHandler(limiter *config.Limiter) context.Handler {
	return func(ctx context.Context) {
		httpError := tollbooth.LimitByRequest(limiter, ctx.Request())
		if httpError != nil {
			ctx.StatusCode(httpError.StatusCode)
			ctx.WriteString(httpError.Message)
			ctx.StopExecution()
			return
		}

		ctx.Next()
	}
}
