package expmetric

import (
	"expvar"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/expmetric.*", "iris-contrib.expmetric")
}

// Handler returns the expvar Iris Handler.
func Handler() iris.Handler {
	return iris.FromStd(expvar.Handler())
}
