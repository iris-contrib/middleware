package raen

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/kataras/iris/v12/context"

	"github.com/getsentry/raven-go"
)

// RecoveryHandler is the iris version of raven-go's `raven.RecoveryHandler`
// https://github.com/getsentry/raven-go/blob/379f8d0a68ca237cf8893a1cdfd4f574125e2c51/http.go#L70.
func RecoveryHandler(ctx context.Context) {
	defer func() {
		if rval := recover(); rval != nil {
			debug.PrintStack()
			rvalStr := fmt.Sprint(rval)
			packet := raven.NewPacket(rvalStr,
				raven.NewException(errors.New(rvalStr),
					raven.NewStacktrace(2, 3, nil)),
				raven.NewHttp(ctx.Request()))
			raven.Capture(packet, nil)
			ctx.StatusCode(500)
		}
	}()

	ctx.Next()
}
