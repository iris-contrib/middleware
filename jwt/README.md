# JSON Web Tokens

**Iris has its own builtin JWT middleware**, which is faster, supports payload encryption, works with the `iris.User` interface and can write and read Go structs or maps as token claims. Prefer it over this community edition: see [kataras/iris/middleware/jwt](https://github.com/kataras/iris/tree/main/middleware/jwt). Note that on Iris v14 the builtin verifier reads tokens from the `Authorization` header only by default; this community middleware keeps its own extractor configuration.

-----

Provides basic JWT functionality for your APIs.
> Full example at: [_example/main.go](_example/main.go)

**1.** Install with `go get github.com/iris-contrib/middleware/jwt@main`

**2.** Import in your code `import "github.com/iris-contrib/middleware/jwt"`

**3.** Define a HS256 secret, e.g. `var mySecret = []byte("My Secret")`. In production you usually load it from a local file or from system environment variables

**4.** Initialize the middleware:

```go
j := jwt.New(jwt.Config{
    ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
        return mySecret, nil
    },
    SigningMethod: jwt.SigningMethodHS256,
})
```

**5.** Generate a token:

```go
token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "foo": "bar",
})

tokenString, _ := token.SignedString(mySecret)
```

**6.** Verify a token with the `j.Serve` middleware before the main handler, e.g.:

```go
app.Get("/protected", j.Serve, protectedHandler)
```

Or per group of routes:

```go
usersAPI := app.Router("/users", j.Serve)
usersAPI.Get("/", protectedHandler)
```

Or inside a handler, with `j.CheckJWT`:

```go
func protectedHandler(ctx iris.Context) {
    if err := j.CheckJWT(ctx); err != nil {
        j.Config.ErrorHandler(ctx, err)
        return
    }

    // [...]
}
```

**7.** Get the verified claims, stored in the Context's `"jwt"` key:

```go
func protectedHandler(ctx iris.Context) {
    // Get the Token verified by the previous handler (of `j.Serve`).
    user := ctx.Values().Get("jwt").(*jwt.Token)

    // A map type of our stored Claims on this verified Token.
    foobar := user.Claims.(jwt.MapClaims)
    for key, value := range foobar {
        ctx.Writef("%s = %s", key, value)
    }
}
```

By default the token is extracted from the `Authorization: Bearer $TOKEN` header. To change this behavior, set a custom `TokenExtractor` in the middleware's configuration. A `TokenExtractor` looks like that:

```go
type TokenExtractor func(iris.Context) (string, error)
```

The package contains some builtin extractors, like `FromParameter`. For example, to accept a token only from a `"token"` URL query parameter:

```go
j := jwt.New(jwt.Config{
    // [...other fields]
    Extractor: jwt.FromParameter("token"),
})
```

You can also combine token extractors by wrapping them with `FromFirst`, e.g. try the "Authorization" header first and the "token" URL query parameter second:

```go
j := jwt.New(jwt.Config{
    // [...other fields]
    Extractor: jwt.FromFirst(jwt.FromAuthHeader, jwt.FromParameter("token")),
})
```
