# cloudwatch

Request latency metrics for [Iris](https://github.com/kataras/iris) v14, sent to [AWS CloudWatch](https://aws.amazon.com/cloudwatch/).

The middleware times each request and puts one `Latency` datum, in microseconds, into a CloudWatch namespace. A handler can put its own metrics through the same client.

## Install

```sh
go get github.com/iris-contrib/middleware/cloudwatch@main
```

```go
import "github.com/iris-contrib/middleware/cloudwatch"
```

The package is named `cloudwatch`, like the AWS SDK package it uses. Import one of the two under another name.

## Example

```go
package main

import (
	"time"

	"github.com/kataras/iris/v14"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	cw "github.com/iris-contrib/middleware/cloudwatch"
)

func main() {
	app := iris.New()

	// The region and the CloudWatch namespace.
	metrics := cw.New("us-east-1", "my-app")
	metrics.ExcludeURLs = []string{"/health"}
	app.Use(metrics.ServeHTTP)

	app.Get("/health", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	app.Get("/", func(ctx iris.Context) {
		// Put a metric of your own next to the latency one.
		put := cw.GetPutFunc(ctx)
		put([]*cloudwatch.MetricDatum{
			{
				MetricName: aws.String("Orders"),
				Timestamp:  aws.Time(time.Now()),
				Unit:       aws.String(cloudwatch.StandardUnitCount),
				Value:      aws.Float64(1),
			},
		})

		ctx.WriteString("success!\n")
	})

	app.Listen(":8080")
}
```

Credentials come from the AWS SDK's default chain: environment variables, the shared credentials file, or the instance role. Without them the page is still served and `NoCredentialProviders` is printed for every metric that could not be sent.

## Fields

`New(region, namespace string) *Cloudwatch` returns a value whose exported fields you may change before the first request.

| Field | Default | Description |
| --- | --- | --- |
| `Service` | a client for the region, 5 retries | The `*cloudwatch.CloudWatch` client that the default `PutMetric` sends through. |
| `Namespace` | the `namespace` argument | The CloudWatch namespace of every metric. |
| `LatencyMetricName` | `"Latency"` | The name of the latency metric. |
| `Before` | `DefaultBefore` | Runs before the rest of the chain. The default stores `PutMetric` on the Context, where `GetPutFunc` finds it. |
| `After` | `DefaultAfter` | Runs after the rest of the chain, with the latency. The default sends the latency metric. |
| `PutMetric` | calls `Service.PutMetricData` | Sends the data. A failure is printed to the standard output. |
| `ExcludeURLs` | none | Request paths that are served without metrics. |

`GetPutFunc(ctx)` returns the put func of the current request, or nil when the middleware did not run for it.

## Things to know before you use it

- The default latency datum has two dimensions, `RequestURI` and `RemoteAddr`. CloudWatch stores, and bills, every distinct combination of dimension values as a separate custom metric. On a public site that is one metric per client address and URL. Set your own `After` to choose the dimensions.
- `PutMetric` is a blocking call to AWS made inside the request, after the handlers have run. Set your own `PutMetric` to batch the data or to send it from another goroutine.
- The module uses `github.com/aws/aws-sdk-go` (v1), which AWS stopped supporting on 31 July 2025. Moving to `aws-sdk-go-v2` would change the types of `Service` and `PutMetric`, so it has not been done here.

## Tests

The tests replace `PutMetric` with a recorder, and one test drives the real CloudWatch client over an in-memory HTTP transport to check the request it builds. Nothing is sent to AWS, so credentials, retries and the real service's answers are not tested.
