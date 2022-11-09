package newrelic

import (
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/newrelic.*", "iris-contrib.newrelic")
}

// headerResponseWriter gives the transaction access to response headers and the
// response code.
type headerResponseWriter struct{ w context.ResponseWriter }

func (w *headerResponseWriter) Header() http.Header       { return w.w.Header() }
func (w *headerResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *headerResponseWriter) WriteHeader(int)           {}

var _ http.ResponseWriter = &headerResponseWriter{}

// replacementResponseWriter mimics the behavior of context.ResponseWriter which
// buffers the response code rather than writing it when
// context.ResponseWriter.WriteHeader is called.
type replacementResponseWriter struct {
	context.ResponseWriter
	replacement http.ResponseWriter
	code        int
	written     bool
}

var _ context.ResponseWriter = &replacementResponseWriter{}

func (w *replacementResponseWriter) flushHeader() {
	if !w.written {
		w.replacement.WriteHeader(w.code)
		w.written = true
	}
}

func (w *replacementResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *replacementResponseWriter) Write(data []byte) (int, error) {
	w.flushHeader()
	return w.ResponseWriter.Write(data)
}

func (w *replacementResponseWriter) FlushResponse() {
	w.flushHeader()
	w.ResponseWriter.FlushResponse()
}

// Context avoids making this package 1.7+ specific.
type Context interface {
	Value(key interface{}) interface{}
}

// TransactionContextKey is used as the context key in
// newrelic.New and newrelic.Transaction. Iris requires
// a string context key. We use two different context keys (and check
// both in newrelic.Transaction and newrelic.FromContext) rather than use a
// single string key because context.WithValue will fail golint if used
// with a string key.
var TransactionContextKey = "newRelicTransaction"

// Transaction returns the transaction stored inside the context, or nil if not
// found.
func Transaction(ctx Context) *newrelic.Transaction {
	if v := ctx.Value(TransactionContextKey); nil != v {
		if txn, ok := v.(*newrelic.Transaction); ok {
			return txn
		}
	}
	return nil
}

// GetTransaction returns the current request's newrelic transaction.
// Kept for backwards compatibility, it calls the Transaction function.
func GetTransaction(ctx iris.Context) *newrelic.Transaction {
	return Transaction(ctx)
}

// Middleware creates an Iris middleware that instruments requests.
//
//	import nriris "github.com/iris-contrib/middleware/newrelic"
//	router := iris.New()
//	// Add the nriris middleware before other middlewares or routes:
//	router.Use(niris.Middleware(app))
func Middleware(app *newrelic.Application) iris.Handler {
	return middleware(app, false)
}

// New calls the Middleware(app). Kept for backwards compatibility.
func New(app *newrelic.Application) iris.Handler {
	return Middleware(app)
}

// MiddleareWithFullPath same as Middleare but
// it sets the option for naming the transaction using
// the registered route path instead of the method + handler name.
func MiddleareWithFullPath(app *newrelic.Application) iris.Handler {
	return middleware(app, true)
}

func middleware(app *newrelic.Application, useFullPath bool) iris.Handler {
	return func(ctx iris.Context) {
		if app != nil {
			w := &headerResponseWriter{w: ctx.ResponseWriter()}
			var nextHandler iris.Handler
			if idx := ctx.HandlerIndex(-1) + 1; idx < len(ctx.Handlers()) {
				nextHandler = ctx.Handlers()[idx]
			}
			if nextHandler == nil { // this should only happen if for some reason the developer added this middleware to the end of the handlers chain.
				return
			}

			var name string
			if useFullPath {
				if route := ctx.GetCurrentRoute(); route != nil {
					name = ctx.Request().Method + " " + ctx.GetCurrentRoute().Tmpl().Src
				} else {
					name = ctx.Request().Method + " " + ctx.FullRequestURI()
				}
			} else {
				name = ctx.Request().Method + " " + context.HandlerName(nextHandler)
			}

			txn := app.StartTransaction(name, newrelic.WithFunctionLocation(nextHandler))
			txn.SetWebRequestHTTP(ctx.Request())
			defer txn.End()

			repl := &replacementResponseWriter{
				ResponseWriter: ctx.ResponseWriter(),
				replacement:    txn.SetWebResponse(w),
				code:           http.StatusOK,
			}
			ctx.ResetResponseWriter(repl)
			defer repl.flushHeader()

			ctx.Values().Set(TransactionContextKey, txn)
		}
		ctx.Next()
	}
}
