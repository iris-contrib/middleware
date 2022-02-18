package expmetric

import (
	"encoding/json"
	"expvar"
	"strconv"

	"github.com/kataras/iris/v12"
)

type (
	// KeyFunc describes a handler which returns a key string.
	KeyFunc func(ctx iris.Context) string
	// ValueFunc describes a handler which returns a value and a boolean which
	// allows or disallows the set of a variable.
	ValueFunc func(ctx iris.Context) (interface{}, bool)
)

type directStringVar string

func (s directStringVar) String() string { return string(s) }

func convertValue(value interface{}) (expvar.Var, error) {
	var valueVar expvar.Var

	switch v := value.(type) {
	case string:
		valueVar = (directStringVar(strconv.Quote(v)))
	case int64:
		valueVar = directStringVar(strconv.Quote(strconv.FormatInt(v, 10)))
	case expvar.Var: // same as fmt.Stringer.
		valueVar = v
	case json.Marshaler:
		b, err := v.MarshalJSON()
		if err != nil {
			return nil, err
		}

		valueVar = directStringVar(string(b))
	default:
		expFunc := func() interface{} {
			return value
		}

		valueVar = expvar.Func(expFunc)
	}

	return valueVar, nil
}

// KeyValue registers a Map expvar which is filled based on
// keyFunc and valueFunc input arguments.
// Supported values that a ValueFunc can return are:
// - string
// - int64
// - any value which completes the String() string method
// - any value which completes the json.Marshaler
// - any struct that can be displayed as JSON
// Look the "convertValue" function for details.
func KeyValue(keyFunc KeyFunc, valueFunc ValueFunc, options ...Option) iris.Handler {
	opts := applyOptions(options)

	if opts.MetricName == "" {
		panic("iris: expmetric: metric name is empty")
	}

	expVar := expvar.NewMap(opts.MetricName)

	return func(ctx iris.Context) {
		if key := keyFunc(ctx); key != "" {
			if value, ok := valueFunc(ctx); ok {
				valueVar, err := convertValue(value)
				if err != nil {
					ctx.SetErr(err)
					ctx.Next()
					return
				}

				expVar.Set(key, valueVar)
			}
		}

		ctx.Next()
	}
}
