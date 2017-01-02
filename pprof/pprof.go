package pprof

import (
	"net/http/pprof"
	"strings"

	"github.com/kataras/iris"
)

// New returns the pprof (profile, debug usage) Handler/ middleware
// NOTE: Route MUST have the last named parameter wildcard named '*action'
// Usage:
// iris.Get("debug/pprof/*action", pprof.New())
func New() iris.HandlerFunc {
	indexHandler := iris.ToHandler(pprof.Index)
	cmdlineHandler := iris.ToHandler(pprof.Cmdline)
	profileHandler := iris.ToHandler(pprof.Profile)
	symbolHandler := iris.ToHandler(pprof.Symbol)
	goroutineHandler := iris.ToHandler(pprof.Handler("goroutine"))
	heapHandler := iris.ToHandler(pprof.Handler("heap"))
	threadcreateHandler := iris.ToHandler(pprof.Handler("threadcreate"))
	debugBlockHandler := iris.ToHandler(pprof.Handler("block"))

	return iris.HandlerFunc(func(ctx *iris.Context) {
		ctx.SetContentType("text/html; charset=" + ctx.Framework().Config.Charset)

		action := ctx.Param("action")
		if len(action) > 1 {
			if strings.Contains(action, "cmdline") {
				cmdlineHandler.Serve((ctx))
			} else if strings.Contains(action, "profile") {
				profileHandler.Serve(ctx)
			} else if strings.Contains(action, "symbol") {
				symbolHandler.Serve(ctx)
			} else if strings.Contains(action, "goroutine") {
				goroutineHandler.Serve(ctx)
			} else if strings.Contains(action, "heap") {
				heapHandler.Serve(ctx)
			} else if strings.Contains(action, "threadcreate") {
				threadcreateHandler.Serve(ctx)
			} else if strings.Contains(action, "debug/block") {
				debugBlockHandler.Serve(ctx)
			}
		} else {
			indexHandler.Serve(ctx)
		}
	})
}
