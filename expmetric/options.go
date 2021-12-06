package expmetric

// Options describes the optional configuration for the expmetric middleware.
type Options struct {
	MetricName string `json:"metric_name" yaml:"MetricName"`
	Resolution int    `json:"resolution" yaml:"Resolution"`
	AvgDiv     int64  `json:"avg_div" yaml:"AvgDiv"`
}

func (opts Options) avgEnabled() bool {
	return opts.AvgDiv > 0
}

type Option func(*Options)

// MetricName sets the name of the metric.
func MetricName(name string) Option {
	return func(opts *Options) {
		opts.MetricName = name
	}
}

// Resolution sets the counter's rate resolution.
func Resolution(n int) Option {
	return func(opts *Options) {
		opts.Resolution = n
	}
}

// Avg sets the dividend. If more than zero then an "$metric_name_avg" expvar is registered.
func Avg(n int64) Option {
	return func(opts *Options) {
		opts.AvgDiv = n
	}
}

func applyOptions(options []Option) (opts Options) {
	for _, fn := range options {
		if fn == nil {
			continue
		}

		fn(&opts)
	}

	if opts.Resolution <= 0 {
		opts.Resolution = 20 // Default value under the hoods of the ratecounter pkg.
	}

	return
}
