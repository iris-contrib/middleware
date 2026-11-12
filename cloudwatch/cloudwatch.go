package cloudwatch

import (
	"fmt"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/cloudwatch.*", "iris-contrib.cloudwatch")
}

// PutMetricContextKey is the context key under which the put func is stored.
const PutMetricContextKey = "PUT_METRIC"

// GetPutFunc returns the put func based on the context,
// this can be called to add more metrics after the middleware's handler is executed.
func GetPutFunc(ctx iris.Context) func([]*cloudwatch.MetricDatum) {
	put := ctx.Values().Get(PutMetricContextKey)
	if put == nil {
		return nil
	}

	if putFunc, ok := put.(func([]*cloudwatch.MetricDatum)); ok {
		return putFunc
	}

	return nil
}

// BeforeFunc called before handler
type BeforeFunc func(iris.Context, *Cloudwatch)

// AfterFunc is the func type called after calling the next func in
// the middleware chain
type AfterFunc func(iris.Context, time.Duration, *Cloudwatch)

// Cloudwatch is the metrics handler.
type Cloudwatch struct {
	// Service is the CloudWatch client the default PutMetric sends through.
	Service *cloudwatch.CloudWatch

	// Namespace is the CloudWatch namespace of every metric sent.
	Namespace string

	// LatencyMetricName is the name of the request latency metric.
	// Defaults to "Latency".
	LatencyMetricName string

	// Before runs before the rest of the chain. Defaults to DefaultBefore.
	Before BeforeFunc
	// After runs after the rest of the chain, with the time it took.
	// Defaults to DefaultAfter.
	After AfterFunc
	// PutMetric sends the data. The default calls Service.PutMetricData
	// and prints a failure to the standard output.
	PutMetric func(data []*cloudwatch.MetricDatum)

	// ExcludeURLs lists the request paths that are served without metrics.
	ExcludeURLs []string
}

// New returns a new *Cloudwatch metrics middleware.
// Register its ServeHTTP method as a handler.
func New(region, namespace string) *Cloudwatch {
	cw := &Cloudwatch{
		Service:           cloudwatch.New(session.New(), aws.NewConfig().WithRegion(region).WithMaxRetries(5)),
		Namespace:         namespace,
		LatencyMetricName: "Latency",
		Before:            DefaultBefore,
		After:             DefaultAfter,
	}
	cw.PutMetric = func(data []*cloudwatch.MetricDatum) {
		putMetric(cw, data)
	}
	return cw
}

func putMetric(cw *Cloudwatch, data []*cloudwatch.MetricDatum) {
	params := &cloudwatch.PutMetricDataInput{
		MetricData: data,
		Namespace:  aws.String(cw.Namespace),
	}
	_, err := cw.Service.PutMetricData(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			fmt.Println(awsErr.Code())
			// if "NoSuchBucket" == awsErr.Code() {
			// 	return *resp
			// }
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func (cw *Cloudwatch) isExcludedURL(s string) bool {
	for i := range cw.ExcludeURLs {
		if cw.ExcludeURLs[i] == s {
			return true
		}
	}

	return false
}

// ServeHTTP is the Iris handler of the middleware.
// It runs Before, the rest of the chain and then After with the latency.
// A request whose path is in ExcludeURLs goes straight to the rest of the chain.
func (cw *Cloudwatch) ServeHTTP(ctx iris.Context) {
	if cw.isExcludedURL(ctx.Request().URL.Path) {
		ctx.Next()
		return
	}

	before, after := cw.Before, cw.After
	if before == nil {
		before = DefaultBefore
	}

	if after == nil {
		after = DefaultAfter
	}

	start := time.Now()

	before(ctx, cw)

	ctx.Next()

	latency := time.Since(start)

	after(ctx, latency, cw)
}

// DefaultBefore is the default func assigned to *Cloudwatch.Before.
// It stores PutMetric on the Context, where GetPutFunc finds it.
func DefaultBefore(ctx iris.Context, cw *Cloudwatch) {
	ctx.Values().Set(PutMetricContextKey, cw.PutMetric)
}

// DefaultAfter is the default func assigned to *Cloudwatch.After.
// It sends the request latency, in microseconds, with the request URI
// and the client address as dimensions.
func DefaultAfter(ctx iris.Context, latency time.Duration, cw *Cloudwatch) {
	us := float64(latency.Nanoseconds()) / float64(time.Microsecond)
	cw.PutMetric([]*cloudwatch.MetricDatum{
		{
			MetricName: aws.String(cw.LatencyMetricName),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("RequestURI"),
					Value: aws.String(ctx.Request().RequestURI),
				},
				{
					Name:  aws.String("RemoteAddr"),
					Value: aws.String(ctx.RemoteAddr()),
				},
			},
			Timestamp: aws.Time(time.Now()),
			Unit:      aws.String("Microseconds"),
			Value:     aws.Float64(us),
		},
	})
	ctx.Values().Remove(PutMetricContextKey)
}
