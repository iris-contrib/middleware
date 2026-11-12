module example_2

go 1.27

require (
	github.com/iris-contrib/middleware/zap v0.0.0
	github.com/kataras/iris/v14 v14.0.0
	go.opentelemetry.io/otel/trace v1.45.0
	go.uber.org/zap v1.28.0
)

require (
	github.com/andybalholm/brotli v1.3.0 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/gomarkdown/markdown v0.0.0-20261109103609-13f93121d67a // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/iris-contrib/schema v0.0.7-0.20250208085038-f68827b0bbfa // indirect
	github.com/kataras/golog v0.2.0 // indirect
	github.com/kataras/httpclient v0.2.0 // indirect
	github.com/kataras/jwt v0.3.0 // indirect
	github.com/klauspost/compress v1.20.4 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/schollz/closestmatch v2.1.0+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/oauth2 v0.38.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// The example always builds against the middleware in this repository.
replace github.com/iris-contrib/middleware/zap => ../../
