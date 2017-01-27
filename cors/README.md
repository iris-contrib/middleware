## Middleware information

This is a very basic cors middleware which allows basic CORS functionality such as allow all origins and allow cradentials.

**If you need more options**, please navigate to [iris-contrib/plugin/cors](https://github.com/iris-conrib/plugin/tree/master/cors).

## How to use

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/iris-contrib/middleware/cors"
)

func main() {
	iris.Use(cors.Default()) // enable all origins, disallow credentials

	iris.Get("/home", func(c *iris.Context) {
		c.Write("Hello from /home")
	})

	iris.Listen(":8080")

}

```

**Change origins and allowCredentials option**

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/iris-contrib/middleware/cors"
)

func main() {
  c := cors.New()
  // to enable credentials headers
  c.AllowCredentials()

  // allow origin per-domain
  c.AddOrigin("http://yourdomainhere.com")
  c.AddOrigin("http://otherdomainhere.com")

	iris.Use(c)

	iris.Get("/home", func(c *iris.Context) {
		c.Write("Hello from /home")
	})

	iris.Listen(":8080")

}

```
