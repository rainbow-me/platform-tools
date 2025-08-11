package observability

import (
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"

	"github.com/rainbow-me/platform-tools/common/logger"
)

type config struct {
	MetricsEnabled   bool
	AnalyticsEnabled bool
	DebugStack       bool
}

type Option func(o *config)

// WithMetrics enables/disables collection of Go Runtime Metrics. Default enabled.
// When enabled, pushes metrics to DataDog every few seconds.
func WithMetrics(enabled bool) Option {
	return func(c *config) {
		c.MetricsEnabled = enabled
	}
}

// WithAnalytics enables/disables trace analytics. Default enabled.
// When enabled, it provides stats about span duration and enables advanced filtering and querying on DataDog, at the
// expense of a slight overhead and slightly increased DataDog cost.
func WithAnalytics(enabled bool) Option {
	return func(c *config) {
		c.AnalyticsEnabled = enabled
	}
}

// WithDebugStack enables/disables capture of stack traces when an error is set on a span. Default disabled.
// If enabled, such stack traces are visible in the DataDog console.
func WithDebugStack(enabled bool) Option {
	return func(c *config) {
		c.DebugStack = enabled
	}
}

// InitObservability initializes our observability stack with sensible defaults that can be overridden.
func InitObservability(serviceName, env string, log *logger.Logger, opts ...Option) {
	log.Info("Starting tracer")
	cfg := &config{
		MetricsEnabled:   true,
		AnalyticsEnabled: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	tracerOpts := []tracer.StartOption{
		tracer.WithEnv(env),
		tracer.WithService(serviceName),
		tracer.WithLogger((*tracerLogger)(log)),
		tracer.WithDebugStack(cfg.DebugStack),
		tracer.WithAnalytics(cfg.AnalyticsEnabled),
	}
	if cfg.MetricsEnabled {
		tracerOpts = append(tracerOpts, tracer.WithRuntimeMetrics())
	}

	if err := tracer.Start(tracerOpts...); err != nil {
		log.Error("Failed to start tracer", logger.Error(err))
	}
	// other observability and telemetry frameworks can be added here if needed
}

type tracerLogger logger.Logger

func (log *tracerLogger) Log(msg string) {
	if log == nil {
		return
	}
	(*logger.Logger)(log).Info(msg)
}
