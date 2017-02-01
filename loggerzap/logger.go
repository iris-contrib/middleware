package loggerzap

import (
	"strconv"
	"time"

	"github.com/kataras/iris"
	"github.com/uber-go/zap"
)

type loggerzapMiddleware struct {
	logger zap.Logger
	config Config
}

// Serve serves the middleware
func (l *loggerzapMiddleware) Serve(ctx *iris.Context) {
	//all except latency to string
	var status, ip, method, path string
	var latency time.Duration
	var startTime, endTime time.Time
	path = ctx.Path()
	method = ctx.Method()

	startTime = time.Now()

	ctx.Next()
	//no time.Since in order to format it well after
	endTime = time.Now()
	latency = endTime.Sub(startTime)

	if l.config.Status {
		status = strconv.Itoa(ctx.ResponseWriter.StatusCode())
	}

	if l.config.IP {
		ip = ctx.RemoteAddr()
	}

	if !l.config.Method {
		method = ""
	}

	if !l.config.Path {
		path = ""
	}

	//finally print the logs
	l.logger.Info("[IRIS] ", zap.String("status", status), zap.Duration("latency", latency), zap.String("ip", ip), zap.String("method", method), zap.String("path", path))

}

// New returns the logger middleware
// receives optional configs(logger.Config)
func New(cfg ...Config) iris.HandlerFunc {
	c := DefaultConfig().Merge(cfg)
	l := &loggerzapMiddleware{config: c}
	l.logger = zap.New(zap.NewTextEncoder())
	return l.Serve
}
