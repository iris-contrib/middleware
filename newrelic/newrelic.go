package newrelic

import (
	"github.com/kataras/iris/context"

	"github.com/newrelic/go-agent"
)

// Newrelic represents the newrelic middleware.
type Newrelic struct {
	Application *newrelic.Application
	Transaction *newrelic.Transaction
}

// Config creates an Config populated with the given appname, license,
// and expected default values.
func Config(applicationName string, licenseKey string) newrelic.Config {
	return newrelic.NewConfig(applicationName, licenseKey)
}

// New creates an middleware and a newrelic.Application and spawns goroutines to manage the
// aggregation and harvesting of data.  On success, a non-nil Newrelic instance and a
// nil error are returned. On failure, a nil Newrelic middleware and a non-nil error
// are returned.
//
// Usage: pass its `ServeHTTP` to a route or globally.
func New(config newrelic.Config) (*Newrelic, error) {
	app, err := newrelic.NewApplication(config)
	return &Newrelic{Application: &app}, err
}

func (n *Newrelic) ServeHTTP(ctx context.Context) {
	r, w := ctx.Request(), ctx.ResponseWriter()
	name := r.URL.Path
	txn := ((*n.Application).StartTransaction(name, w, r)).(newrelic.Transaction)
	n.Transaction = &txn
	defer (*n.Transaction).End()

	ctx.Next()
}
