## Repository information

This Repository contains all middleware for the [Iris web framework](https://github.com/kataras/iris).

You can contribute also, just make a pull request, try to keep conversion, configuration file: './mymiddleware/config.go' & middleware: './mymiddleware/mymiddleware.go'.


Middleware is just handler(s) which served before the main handler.


## How can I install a middleware?

```sh
$ go get -u github.com/iris-contrib/middleware/$FOLDERNAME
```

**NOTE**: When you install one middleware you will have all of them downloaded & installed, **no need** to re-run the go get foreach middeware.

## How can I register middleware?


**To a single route**
```go
iris.Get("/mypath",myMiddleware1,myMiddleware2,func(ctx *iris.Context){}, func(ctx *iris.Context){},myMiddleware5,myMainHandlerLast)
```

**To a party of routes or subdomain**
```go

myparty := iris.Party("/myparty", myMiddleware1,func(ctx *iris.Context){},myMiddleware3)
{
	//....
}

```

**To all routes**
```go
iris.UseFunc(func(ctx *iris.Context){}, myMiddleware2)
```

**To global, all routes on all subdomains on all parties**
```go
iris.UseGlobalFunc(func(ctx *iris.Context){}, myMiddleware2)
```

# Can I use standard net/http handler with Iris?

**Yes** you can, just pass the Handler inside the `iris.ToHandler` in order to be converted into iris.HandlerFunc and register it as you saw before. 

## handler which has the form of http.Handler/HandlerFunc

```go
package main

import (
	"github.com/kataras/iris"
)

func main() {

	sillyHTTPHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	     println(r.RequestURI)
	})
	
	sillyConvertedToIris := iris.ToHandler(sillyHTTPHandler)
	iris.Use(sillyConvertedToIris)

	iris.Listen(":8080")
}

```


## middleware which wraps http.Handler and returns a modified http.Handler: 

### From net/http:

```go
package main

import (
    "net/http"
    "github.com/rs/cors"
)

func main() {

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte("{\"hello\": \"world\"}"))
    })

    // cors.Default() setup the middleware with default options being
    // all origins accepted with simple methods (GET, POST). See
    // documentation below for more options.
    handler := cors.Default().Handler(mux)
    http.ListenAndServe(":8080", handler)
}

```

### TO IRIS

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/rs/cors"
)

func main() {

    iris.Get("/", func(ctx *iris.Context){
       ctx.SetContentType( "application/json")
       ctx.Write([]byte("{\"hello\": \"world\"}"))
    })
    
    // special case when you need to use a middleware that wraps 
    // the whole mux, you just override the iris' Router:
    iris.Router = cors.Default().Handler(iris.Router)
    
    iris.Listen(":8080")
}

```

# What if, the middleware contains a `next http.Handler` ?

Easy, just convert the middleware's source code to iris-compatible, and if you like make a PR to this repository to share it with the community:

1. replace all method's parameters(and the third parameter `next http.Handler`) `with ctx *iris.Context`. **Simple**
2. inside the handler or its ServeHTTP method, replace the `next(w,r)` with `ctx.Next()`. **Readability**
3. register the middleware as you saw before. **Clean**

Example:

```go
// let's say that the middleware you want to use with iris, has this form:

func (a *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	
	if r.RequestURI != "/denied" {
	   next(w,r)
	   return
	}
	
	w.WriteHeader(http.StatusNotAllowed)
	w.Write([]byte("You are not allowed to be here!"))
}

/* do some tricky chain things or wrap it to another http.Handler, whatever, forget these things when you use Iris, see below...*/

```
```go
// let's convert it to iris, see how simple it is:

// *note: Serve is the iris.Handler signature, but you can skip it and call it manually as iris.HandlerFunc
func (a *middleware) Serve(ctx *iris.Context){ // 1. simple

	if ctx.Path() != "/denied" {
	   ctx.Next() // 2. readability
	   return
	}
	
	ctx.Text(iris.StatusNotAllowed, "You are not allowed to be here!")
})

myMiddleware := &middleware{}
iris.Use(myMiddleware) // 3. clean 
```

> * or iris.useFunc(myMiddleware.Serve)
