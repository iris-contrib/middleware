package cloudwatch

import (
	"fmt"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/cloudwatch.*", "iris-contrib.cloudwatch")
}

// PutMetricContextKey is the context key which the metrics are stored.
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
	// CloudWatch underline Service
	Service *cloudwatch.CloudWatch

	// CloudWatch underline namespace
	Namespace string

	// Latency underlime metric name
	LatencyMetricName string

	Before    BeforeFunc
	After     AfterFunc
	PutMetric func(data []*cloudwatch.MetricDatum)

	// ExcludeURLs from logging
	ExcludeURLs []string
}

// New retruns a new *Cloudwatch metrics middleware
// call its ServeHTTP to adapt it on the chain.
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

func (cw *Cloudwatch) ServeHTTP(ctx iris.Context) {
	if cw.Before == nil {
		cw.Before = DefaultBefore
	}

	if cw.After == nil {
		cw.After = DefaultAfter
	}
	r := ctx.Request()

	if ok := cw.isExcludedURL(r.URL.Path); ok {
		return
	}

	start := time.Now()

	cw.Before(ctx, cw)

	ctx.Next()

	latency := time.Since(start)

	cw.After(ctx, latency, cw)
}

// DefaultBefore is the default func assigned to *Cloudwatch.Before
func DefaultBefore(ctx iris.Context, cw *Cloudwatch) {
	ctx.Values().Set(PutMetricContextKey, cw.PutMetric)
}

// DefaultAfter is the default func assigned to *Cloudwatch.After
func DefaultAfter(ctx iris.Context, latency time.Duration, cw *Cloudwatch) {
	ms := float64(latency.Nanoseconds() * 1000)
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
			Value:     aws.Float64(ms),
		},
	})
	ctx.Values().Remove(PutMetricContextKey)
}
