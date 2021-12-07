package expmetric

import (
	"expvar"
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/paulbellamy/ratecounter"
)

// HitsPerHour registers an expvar counter which increments
// the number of hits for the last minute.
func HitsPerMinute(options ...Option) iris.Handler {
	opts := applyOptions(options)

	if opts.MetricName == "" {
		opts.MetricName = "hits_per_minute"
	}

	opts.AvgDiv = 60

	return hits(time.Minute, opts)
}

// HitsPerHour registers an expvar counter which increments
// the number of hits for the last second.
func HitsPerSecond(options ...Option) iris.Handler {
	opts := applyOptions(options)

	if opts.MetricName == "" {
		opts.MetricName = "hits_per_second"
	}

	return hits(time.Second, opts)
}

// HitsPerHour registers an expvar counter which increments
// the number of hits for the last hour.
func HitsPerHour(options ...Option) iris.Handler {
	opts := applyOptions(options)

	if opts.MetricName == "" {
		opts.MetricName = "hits_per_hour"
	}

	opts.AvgDiv = 24

	return hits(time.Hour, opts)
}

// HitsTotal registers an expvar counter which increments
// the number of hits.
func HitsTotal(options ...Option) iris.Handler {
	opts := applyOptions(options)

	if opts.MetricName == "" {
		opts.MetricName = "hits_total"
	}

	var counter ratecounter.Counter

	hitsVar := expvar.NewInt(opts.MetricName)

	var hitsAvgVar *expvar.Int
	if opts.avgEnabled() {
		hitsAvgVar = expvar.NewInt(getMetricNameForAvg(opts.MetricName))
	}

	return func(ctx iris.Context) {
		counter.Incr(1)
		value := counter.Value()
		hitsVar.Set(value)
		if opts.avgEnabled() {
			hitsAvgVar.Set(value / opts.AvgDiv)
		}

		ctx.Next()
	}
}

// HitsTotalWithKeyFunc registers a Map expvar counter which increments
// the number of hits inside a map based on the keyFunc.
func HitsTotalWithKeyFunc(keyFunc func(ctx iris.Context) string, options ...Option) iris.Handler {
	opts := applyOptions(options)

	if opts.MetricName == "" {
		panic("iris: expmetric: metric name is empty")
	}

	hitsVar := expvar.NewMap(opts.MetricName)

	return func(ctx iris.Context) {
		if key := keyFunc(ctx); key != "" {
			hitsVar.Add(key, 1)
		}

		ctx.Next()
	}
}

// HitsTotalPerUserEmail registers a Map expvar counter which increments
// the number of hits per authenticated user's email.
func HitsTotalPerUserEmail(options ...Option) iris.Handler {
	keyFunc := func(ctx iris.Context) string {
		email, _ := ctx.User().GetEmail()
		return email
	}

	return HitsTotalWithKeyFunc(keyFunc, options...)
}

func getMetricNameForAvg(metricName string) string {
	charSplitter := "_"
	if strings.Contains(metricName, ".") {
		charSplitter = "."
	}

	return fmt.Sprintf("%s%savg", metricName, charSplitter)
}

func hits(interval time.Duration, opts Options) iris.Handler {
	if interval <= 0 {
		panic("iris: expmetric: interval zero or less")
	}

	if opts.MetricName == "" {
		panic("iris: expmetric: metric name is empty")
	}

	if opts.AvgDiv <= 0 {
		if interval.Seconds() == 60 {
			opts.AvgDiv = 60
		} else if interval.Hours() == 24 {
			opts.AvgDiv = 24
		}
	}

	hitsVar := expvar.NewInt(opts.MetricName)

	var hitsAvgVar *expvar.Int
	if opts.avgEnabled() {
		hitsAvgVar = expvar.NewInt(getMetricNameForAvg(opts.MetricName))
	}

	counter := ratecounter.NewRateCounter(interval).WithResolution(opts.Resolution)
	counter.OnStop(func(f *ratecounter.RateCounter) {
		hitsVar.Set(0)
		if opts.avgEnabled() {
			hitsAvgVar.Set(0)
		}
	})

	return func(ctx iris.Context) {
		counter.Incr(1)
		value := counter.Rate()
		hitsVar.Set(value)
		if opts.avgEnabled() {
			hitsAvgVar.Set(value / opts.AvgDiv)
		}

		ctx.Next()
	}
}
