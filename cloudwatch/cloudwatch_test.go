package cloudwatch_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	cw "github.com/iris-contrib/middleware/cloudwatch"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// recorder is a fake PutMetric: it keeps what the middleware would have sent.
type recorder struct {
	mu   sync.Mutex
	data []*cloudwatch.MetricDatum
}

func (r *recorder) put(data []*cloudwatch.MetricDatum) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = append(r.data, data...)
}

func (r *recorder) all() []*cloudwatch.MetricDatum {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*cloudwatch.MetricDatum(nil), r.data...)
}

// newMiddleware returns the middleware with its PutMetric replaced by a recorder,
// so nothing reaches AWS.
func newMiddleware() (*cw.Cloudwatch, *recorder) {
	m := cw.New("us-east-1", "test")
	rec := new(recorder)
	m.PutMetric = rec.put
	return m, rec
}

func dimensions(d *cloudwatch.MetricDatum) map[string]string {
	m := make(map[string]string, len(d.Dimensions))
	for _, dim := range d.Dimensions {
		m[aws.StringValue(dim.Name)] = aws.StringValue(dim.Value)
	}
	return m
}

func TestLatencyMetric(t *testing.T) {
	m, rec := newMiddleware()

	const sleep = 20 * time.Millisecond

	app := iris.New()
	app.Use(m.ServeHTTP)
	app.Get("/slow", func(ctx iris.Context) {
		time.Sleep(sleep)
		ctx.WriteString("done")
	})

	e := httptest.New(t, app)
	e.GET("/slow").WithQuery("a", "b").Expect().Status(httptest.StatusOK).Body().IsEqual("done")

	data := rec.all()
	if len(data) != 1 {
		t.Fatalf("expected one metric datum, got %d", len(data))
	}

	d := data[0]
	if got := aws.StringValue(d.MetricName); got != "Latency" {
		t.Fatalf("expected the default metric name Latency, got %q", got)
	}

	if got := aws.StringValue(d.Unit); got != cloudwatch.StandardUnitMicroseconds {
		t.Fatalf("expected the unit %s, got %q", cloudwatch.StandardUnitMicroseconds, got)
	}

	// The value is in the unit it is labelled with: at least the time the
	// handler slept and nowhere near a minute.
	got := time.Duration(aws.Float64Value(d.Value)) * time.Microsecond
	if got < sleep || got > time.Minute {
		t.Fatalf("expected a latency between %s and %s, got %s", sleep, time.Minute, got)
	}

	dims := dimensions(d)
	if dims["RequestURI"] != "/slow?a=b" {
		t.Fatalf("expected the RequestURI dimension /slow?a=b, got %q", dims["RequestURI"])
	}

	if _, ok := dims["RemoteAddr"]; !ok {
		t.Fatal("expected a RemoteAddr dimension")
	}

	if d.Timestamp == nil || time.Since(*d.Timestamp) > time.Minute {
		t.Fatalf("expected a current timestamp, got %v", d.Timestamp)
	}
}

func TestLatencyMetricName(t *testing.T) {
	m, rec := newMiddleware()
	m.LatencyMetricName = "ResponseTime"

	app := iris.New()
	app.Use(m.ServeHTTP)
	app.Get("/", func(ctx iris.Context) {})

	httptest.New(t, app).GET("/").Expect().Status(httptest.StatusOK)

	data := rec.all()
	if len(data) != 1 || aws.StringValue(data[0].MetricName) != "ResponseTime" {
		t.Fatalf("expected one ResponseTime datum, got %v", data)
	}
}

// A handler adds its own metrics through GetPutFunc.
func TestGetPutFunc(t *testing.T) {
	m, rec := newMiddleware()

	app := iris.New()
	app.Get("/with", m.ServeHTTP, func(ctx iris.Context) {
		put := cw.GetPutFunc(ctx)
		if put == nil {
			ctx.StopWithText(iris.StatusInternalServerError, "no put func")
			return
		}

		put([]*cloudwatch.MetricDatum{{
			MetricName: aws.String("Orders"),
			Unit:       aws.String(cloudwatch.StandardUnitCount),
			Value:      aws.Float64(42),
		}})
	})
	// Without the middleware there is nothing to put with.
	app.Get("/without", func(ctx iris.Context) {
		if cw.GetPutFunc(ctx) != nil {
			ctx.StopWithText(iris.StatusInternalServerError, "unexpected put func")
		}
	})

	e := httptest.New(t, app)
	e.GET("/with").Expect().Status(httptest.StatusOK)
	e.GET("/without").Expect().Status(httptest.StatusOK)

	data := rec.all()
	if len(data) != 2 {
		t.Fatalf("expected the handler's metric and the latency, got %d data", len(data))
	}

	if aws.StringValue(data[0].MetricName) != "Orders" || aws.Float64Value(data[0].Value) != 42 {
		t.Fatalf("expected the handler's metric first, got %v", data[0])
	}

	if aws.StringValue(data[1].MetricName) != "Latency" {
		t.Fatalf("expected the latency metric last, got %v", data[1])
	}
}

// An excluded path is served like any other, without metrics.
func TestExcludeURLs(t *testing.T) {
	m, rec := newMiddleware()
	m.ExcludeURLs = []string{"/health"}

	app := iris.New()
	app.Use(m.ServeHTTP)
	app.Get("/health", func(ctx iris.Context) { ctx.WriteString("ok") })
	app.Get("/work", func(ctx iris.Context) { ctx.WriteString("worked") })

	e := httptest.New(t, app)
	e.GET("/health").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
	if n := len(rec.all()); n != 0 {
		t.Fatalf("expected no metrics for an excluded path, got %d", n)
	}

	e.GET("/work").Expect().Status(httptest.StatusOK).Body().IsEqual("worked")
	if n := len(rec.all()); n != 1 {
		t.Fatalf("expected one metric for a path that is not excluded, got %d", n)
	}
}

func TestBeforeAndAfter(t *testing.T) {
	m, rec := newMiddleware()

	var order []string
	m.Before = func(ctx iris.Context, c *cw.Cloudwatch) {
		order = append(order, "before")
	}
	m.After = func(ctx iris.Context, latency time.Duration, c *cw.Cloudwatch) {
		if latency < 0 {
			t.Errorf("expected a latency of zero or more, got %s", latency)
		}
		order = append(order, "after")
	}

	app := iris.New()
	app.Get("/", m.ServeHTTP, func(ctx iris.Context) {
		order = append(order, "handler")
	})

	httptest.New(t, app).GET("/").Expect().Status(httptest.StatusOK)

	if len(order) != 3 || order[0] != "before" || order[1] != "handler" || order[2] != "after" {
		t.Fatalf("expected before, handler, after but got %v", order)
	}

	if n := len(rec.all()); n != 0 {
		t.Fatalf("expected a custom After to send nothing, got %d data", n)
	}
}

// roundTripper answers the CloudWatch API in memory and keeps the form it was sent.
type roundTripper struct {
	mu    sync.Mutex
	forms []url.Values
}

func (rt *roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	var body io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		body = zr
	}

	b, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	form, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, err
	}

	rt.mu.Lock()
	rt.forms = append(rt.forms, form)
	rt.mu.Unlock()

	const response = `<PutMetricDataResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">` +
		`<ResponseMetadata><RequestId>test</RequestId></ResponseMetadata></PutMetricDataResponse>`

	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{"Content-Type": []string{"text/xml"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(response))),
		Request:    r,
	}, nil
}

// The default PutMetric sends the data through Service with the Namespace.
// The CloudWatch client is real; only its HTTP transport is a fake.
func TestDefaultPutMetricThroughService(t *testing.T) {
	rt := new(roundTripper)

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials("AKIATEST", "secret", ""),
		HTTPClient:  &http.Client{Transport: rt},
		MaxRetries:  aws.Int(0),
	})
	if err != nil {
		t.Fatal(err)
	}

	m := cw.New("us-east-1", "MyNamespace")
	m.Service = cloudwatch.New(sess)

	app := iris.New()
	// The in-memory test client has no address, and CloudWatch does not
	// accept an empty dimension value, so give the request one.
	app.UseRouter(func(ctx iris.Context) {
		ctx.Request().RemoteAddr = "192.0.2.10:54321"
		ctx.Next()
	})
	app.Use(m.ServeHTTP)
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })

	httptest.New(t, app).GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")

	rt.mu.Lock()
	defer rt.mu.Unlock()

	if len(rt.forms) != 1 {
		t.Fatalf("expected one request to CloudWatch, got %d", len(rt.forms))
	}

	form := rt.forms[0]
	for key, expected := range map[string]string{
		"Action":                         "PutMetricData",
		"Namespace":                      "MyNamespace",
		"MetricData.member.1.MetricName": "Latency",
		"MetricData.member.1.Unit":       "Microseconds",
		"MetricData.member.1.Dimensions.member.1.Name":  "RequestURI",
		"MetricData.member.1.Dimensions.member.1.Value": "/",
		"MetricData.member.1.Dimensions.member.2.Name":  "RemoteAddr",
		"MetricData.member.1.Dimensions.member.2.Value": "192.0.2.10",
	} {
		if got := form.Get(key); got != expected {
			t.Errorf("expected the form value %s=%q, got %q", key, expected, got)
		}
	}
}
