# This is a middleware ported to Iris

## Link: [https://github.com/geekypanda/oauth2server](https://github.com/geekypanda/oauth2server)

This snippet shows how to create an authorization server
```go
package main

import (
	"time"

	"github.com/geekypanda/oauth2server"
	"github.com/kataras/iris"
)

func main() {
  s := oauth2server.NewOAuthBearerServer(
		"mySecretKey-10101",
		time.Second*120,
		&TestUserVerifier{},
		nil)
	iris.Post("/token", s.UserCredentials)
	iris.Post("/auth", s.ClientCredentials)

	iris.Listen(":9090")
}
```

This snippet shows how to use the middleware
```go
    authorized := iris.Party("/authorized")
	// use the Bearer Athentication middleware
	authorized.Use(oauth2server.Authorize("mySecretKey-10101", nil))

	authorized.Get("/customers", GetCustomers)
	authorized.Get("/customers/:id/orders", GetOrders)
```

