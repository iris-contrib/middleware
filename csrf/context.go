package csrf

import (
	"fmt"

	"github.com/kataras/iris/v12"
)

func contextGet(ctx iris.Context, key string) (interface{}, error) {
	val := ctx.Values().Get(key)
	if val == nil {
		return nil, fmt.Errorf("no value exists in the context for key %q", key)
	}

	return val, nil
}

func contextSave(ctx iris.Context, key string, val interface{}) {
	ctx.Values().Set(key, val)
}
