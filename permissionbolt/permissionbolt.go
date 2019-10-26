/*
// Doesn't work and can't be fixed within our repository
// because of upstream author of permissionbolt wrong use of its vendor folder:
// permissionbolt\permissionbolt.go:22:2: cannot use perm (type *permissionbolt.Permissions) as type "github.com/xyproto/pinterface".IPermissions in return argument:
//         *permissionbolt.Permissions does not implement "github.com/xyproto/pinterface".IPermissions (wrong type for UserState method)
//                 have UserState() "github.com/xyproto/permissionbolt/vendor/github.com/xyproto/pinterface".IUserState
// 				want UserState() "github.com/xyproto/pinterface".IUserState

import (
	"github.com/kataras/iris/v12/context"
	"github.com/xyproto/permissionbolt"
	"github.com/xyproto/pinterface"
)

// New returns an new permissionbolt struct (that satisfies
// `pinterface.IPermissions`), a new iris.Handler function that can deny
// unauthorized access to a set of URL prefixes and an error if things go
// wrong. `pinterface.IPermissions` is used instead of
// `*permissions.Permissions` in order to be compatible with not only
// `permissionbolt`, but also other database backends, like for example
// `permissions2`, which uses Redis.
func New(filename string) (pinterface.IPermissions, context.Handler, error) {
	perm, err := permissionbolt.NewWithConf(filename)
	if err != nil {
		return nil, nil, err
	}
	// Return the permissions struct together with an Iris middleware Handler
	return perm, func(ctx context.Context) {
		// Check if the user has the right admin/user rights
		if perm.Rejected(ctx.ResponseWriter(), ctx.Request()) {
			// Stop the request for executing further
			ctx.StopExecution()
			// Let the user know, by calling the custom "permission denied" function
			perm.DenyFunction()(ctx.ResponseWriter(), ctx.Request())
			return
		}
		// Serve the next handler if permissions were granted
		ctx.Next()
	}, nil
}
*/
package permissions
