package prometheus

import (
	"strconv"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/prometheus.*", "iris-contrib.prometheus")
}

var (
	// DefaultBuckets prometheus buckets in seconds.
	DefaultBuckets = []float64{0.3, 1.2, 5.0}
)

const (
	reqsName    = "http_requests_total"
	latencyName = "http_request_duration_seconds"
)

// Prometheus is a handler that records prometheus metrics for the number of requests
// and their latency, partitioned by status code, method and HTTP path.
//
// Usage: pass its `ServeHTTP` to a route or globally.
type Prometheus struct {
	reqs    *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

// New returns a new prometheus middleware and registers its two collectors,
// http_requests_total and http_request_duration_seconds, to the default
// prometheus registry. The "name" is the value of their "service" label.
// It panics if it is called a second time with the same name.
//
// If buckets are empty then `DefaultBuckets` are set.
func New(name string, buckets ...float64) *Prometheus {
	p := Prometheus{}
	p.reqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        reqsName,
			Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
			ConstLabels: prometheus.Labels{"service": name},
		},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(p.reqs)

	if len(buckets) == 0 {
		buckets = DefaultBuckets
	}

	p.latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        latencyName,
		Help:        "How long it took to process the request, partitioned by status code, method and HTTP path.",
		ConstLabels: prometheus.Labels{"service": name},
		Buckets:     buckets,
	},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(p.latency)

	return &p
}

// ServeHTTP is the Iris handler of the middleware. It runs the rest of the
// chain and then records the request with the status code it ended with.
func (p *Prometheus) ServeHTTP(ctx iris.Context) {
	start := time.Now()
	ctx.Next()
	r := ctx.Request()
	statusCode := strconv.Itoa(ctx.GetStatusCode())

	p.reqs.WithLabelValues(statusCode, r.Method, r.URL.Path).
		Inc()

	p.latency.WithLabelValues(statusCode, r.Method, r.URL.Path).
		Observe(float64(time.Since(start).Nanoseconds()) / 1000000000)
}
