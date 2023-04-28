// Package iriszap provides log handling using zap package.
package iriszap

import (
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/kataras/iris/v12"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Fn func(ctx iris.Context) []zapcore.Field

// Config is config setting for iriszap.
type Config struct {
	TimeFormat string
	UTC        bool
	SkipPaths  []string
	Context    Fn
}

// New returns an iris.Handler (middleware) that logs requests using uber-go/zap.
//
// Requests with errors are logged using zap.Error().
// Requests without errors are logged using zap.Info().
//
// It receives:
//  1. A time package format string (e.g. time.RFC3339).
//  2. A boolean stating whether to use UTC time zone or local.
func New(logger *zap.Logger, timeFormat string, utc bool) iris.Handler {
	return NewWithConfig(logger, &Config{TimeFormat: timeFormat, UTC: utc})
}

// NewWithConfig returns an iris.Handler using configs.
func NewWithConfig(logger *zap.Logger, conf *Config) iris.Handler {
	skipPaths := make(map[string]bool, len(conf.SkipPaths))
	for _, path := range conf.SkipPaths {
		skipPaths[path] = true
	}

	return func(ctx iris.Context) {
		start := time.Now()
		// some evil middlewares modify this values
		path := ctx.Request().URL.Path
		query := ctx.Request().URL.RawQuery
		ctx.Next()

		if _, ok := skipPaths[path]; !ok {
			end := time.Now()
			latency := end.Sub(start)
			if conf.UTC {
				end = end.UTC()
			}

			fields := []zapcore.Field{
				zap.Int("status", ctx.GetStatusCode()),
				zap.String("method", ctx.Method()),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", ctx.RemoteAddr()),
				zap.String("user-agent", ctx.Request().UserAgent()),
				zap.Duration("latency", latency),
			}
			if conf.TimeFormat != "" {
				fields = append(fields, zap.String("time", end.Format(conf.TimeFormat)))
			}

			if conf.Context != nil {
				fields = append(fields, conf.Context(ctx)...)
			}

			if ctx.GetErr() != nil {
				// Log error field if this is an erroneous request.
				logger.Error(ctx.GetErr().Error(), fields...)
			} else {
				logger.Info(path, fields...)
			}
		}
	}
}

func defaultHandleRecovery(ctx iris.Context, err interface{}) {
	ctx.StopWithStatus(iris.StatusInternalServerError)
}

// RecoveryWithZap returns an iris.Handler (middleware)
// that recovers from any panics and logs requests using uber-go/zap.
// All errors are logged using zap.Error().
// stack means whether output the stack info.
// The stack info is easy to find where the error occurs but the stack info is too large.
func RecoveryWithZap(logger *zap.Logger, stack bool) iris.Handler {
	return CustomRecoveryWithZap(logger, stack, defaultHandleRecovery)
}

// CustomRecoveryWithZap returns an iris.Handler (middleware) with a custom recovery handler
// that recovers from any panics and logs requests using uber-go/zap.
// All errors are logged using zap.Error().
// stack means whether output the stack info.
// The stack info is easy to find where the error occurs but the stack info is too large.
func CustomRecoveryWithZap(logger *zap.Logger, stack bool, recovery func(iris.Context, interface{})) iris.Handler {
	return func(ctx iris.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(ctx.Request(), false)
				if brokenPipe {
					logger.Error(ctx.Request().URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					ctx.SetErr(err.(error)) // nolint: errcheck
					ctx.StopExecution()
					return
				}

				if stack {
					logger.Error("[Recovery from panic]",
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					logger.Error("[Recovery from panic]",
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				recovery(ctx, err)
			}
		}()

		ctx.Next()
	}
}
