## Middleware information

This folder contains a middleware which loggs http-requests using [zap](https://github.com/uber-go/zap). Why zap? Zap promises to be faster than default golang log.


## Install

```sh
$ go get -u github.com/iris-contrib/middleware/loggerzap
```

**Logs the incoming requests**

## How to use

It's used similar to standard logger, just replace with loggerzap. Read the logger section [here](https://kataras.gitbooks.io/iris/content/logger.html)
