# Casbin

[Iris](https://github.com/kataras/iris) web framework's [casbin](https://github.com/casbin/casbin) middleware.

The authorization determines a request based on `{subject, object, action}`. Read [Casbin's documentation](https://github.com/casbin/casbin) first to understand how the model and policy files work.

```sh
$ go get github.com/casbin/casbin/v2@latest
$ go get github.com/iris-contrib/middleware/casbin@main
```

## Table of contents

- [UseRouter: recommended, provides the full casbin functionality](_examples/router/main.go)
- [Use: register to specific routes or routers](_examples/middleware/main.go)

> Each example has its own model, configuration and tests, please read them as well.
