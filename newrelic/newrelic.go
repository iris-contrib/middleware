package newrelic

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// Migration from newrelic to newrelic/v3:
// https://github.com/newrelic/go-agent/blob/master/MIGRATION.md#configuration

// Aliases for the two most common options.
var (
	// ConfigAppName sets the application name.
	ConfigAppName = newrelic.ConfigAppName

	// ConfigLicense sets the license.
	ConfigLicense = newrelic.ConfigLicense
)

// AppConnectTimeout is the time used on `New` function in order to wajit for the application to connect.
// Defaults to 5 seconds.
var AppConnectTimeout = 5 * time.Second

// New accepts the newrelic options and returns a new middleware for newrelic.
// Example: New(ConfigAppName("My Application"), ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")))
// Note that the Context's response writer's underline writer will be upgraded to the newrelic's one.
// Look `GetTransaction` to retrieve the transaction created.
// Use `Wrap` instead when you have an existing newrelic Application instance.
func New(opts ...newrelic.ConfigOption) (iris.Handler, error) {
	app, err := newrelic.NewApplication(opts...)
	if err != nil {
		return nil, err
	}

	// Wait for the application to connect.
	if err = app.WaitForConnection(AppConnectTimeout); nil != err {
		return nil, err
	}

	return Wrap(app), nil
}

const transactionContextKey = "iris.newrelic.transaction"

// Wrap accepts an existing newrelic Application and returns its Iris middleware.
// Note that the Context's response writer's underline writer will be upgraded to the newrelic's one.
// See `GetTransaction` to retrieve the transaction created.
func Wrap(app *newrelic.Application) iris.Handler {
	return func(ctx iris.Context) {
		name := ctx.Path()
		txn := app.StartTransaction(name)
		defer txn.End()

		txn.SetWebRequestHTTP(ctx.Request())
		ctx.Values().Set(transactionContextKey, txn)

		txnWriter := txn.SetWebResponse(ctx.ResponseWriter())
		ctx.ResponseWriter().SetWriter(txnWriter)

		ctx.Next()
	}
}

// GetTransaction returns the current request's newrelic Transaction.
func GetTransaction(ctx iris.Context) *newrelic.Transaction {
	if v := ctx.Values().Get(transactionContextKey); v != nil {
		if t, ok := v.(*newrelic.Transaction); ok {
			// No need to return its writer, context has been modified.
			// if v = ctx.Values().Get(transactionWriterContextKey); v != nil {
			// 	if w, ok := v.(http.ResponseWriter); ok {
			// 		return t, w
			// 	}
			// }
			return t
		}
	}

	return nil
}
