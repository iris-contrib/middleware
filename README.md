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
iris.MustUseFunc(func(ctx *iris.Context){}, myMiddleware2)
```
