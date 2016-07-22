package recovery

import (
	"runtime/debug"
	"time"

	"github.com/iris-contrib/logger"
	"github.com/kataras/iris"
)

type recovery struct {
	// logger which outputs the panic messages
	logger *logger.Logger
}

func (r *recovery) Serve(ctx *iris.Context) {
	defer func() {
		if err := recover(); err != nil {
			r.logger.Dangerf(time.Now().Format(logger.TimeFormat)+": Recovery from panic\n%s\n%s\n", err, debug.Stack())
			//ctx.Panic just sends  http status 500 by default, but you can change it by: iris.OnPanic(func( c *iris.Context){})
			ctx.Panic()
		}
	}()
	ctx.Next()
}

// New restores the server on internal server errors (panics)
// receives an optional logger, the default is the Logger with an os.Stderr as its output
// returns the middleware
func New(lgs ...*logger.Logger) iris.HandlerFunc {
	/*r := recovery{os.Stderr}
	if out != nil && len(out) == 1 {
		r.out = out[0]
	}*/
	log := logger.New(logger.DefaultConfig())
	if len(lgs) > 0 {
		log = lgs[0]
	}
	r := &recovery{logger: log}
	return r.Serve
}
