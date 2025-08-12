package observability

import (
	"regexp"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/samber/lo"

	"github.com/rainbow-me/platform-tools/common/logger"
)

type config struct {
	MetricsEnabled   bool
	AnalyticsEnabled bool
	DebugStack       bool
	SamplingRate     float64
	SamplingRules    []SamplingRule
}

type SamplingRule struct {
	// OpName specifies the regex pattern that a span operation name must match.
	OpName *regexp.Regexp

	// Resource specifies the regex pattern that a span resource must match.
	Resource *regexp.Regexp

	// Rate specifies the sampling rate that should be applied to spans that match service and/or name of the rule.
	Rate float64

	// MaxPerSecond specifies max number of spans per second that can be sampled per the rule.
	// If not specified, the default is no limit.
	MaxPerSecond float64
}

func convertRule(sr SamplingRule, _ int) tracer.SamplingRule {
	return tracer.SamplingRule{
		Name:         sr.OpName,
		Rate:         sr.Rate,
		MaxPerSecond: sr.MaxPerSecond,
		Resource:     sr.Resource,
	}
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

// WithSamplingRate specifies the percentage of traces that will actually be sent. Must be between 0 and 1 (0-100%).
// By default, an all-permissive sampler rate (1) is used.
func WithSamplingRate(rate float64) Option {
	return func(c *config) {
		c.SamplingRate = rate
	}
}

// WithSamplingRules allow specifying more sophisticated per-span sampling rates.
func WithSamplingRules(rules []SamplingRule) Option {
	return func(c *config) {
		c.SamplingRules = rules
	}
}

type StopFunc func()

// InitObservability initializes our observability stack with sensible defaults that can be overridden.
// Don't forget to add a shutdown hook for deferred calling of the returned StopFunc.
func InitObservability(serviceName, env string, log *logger.Logger, opts ...Option) StopFunc {
	log.Info("Starting tracer")
	cfg := &config{
		MetricsEnabled:   true,
		AnalyticsEnabled: true,
		DebugStack:       false,
		SamplingRules:    []SamplingRule{},
		SamplingRate:     1.0,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	tracerOpts := []tracer.StartOption{
		tracer.WithEnv(env),
		tracer.WithService(serviceName),
		tracer.WithLogger((*logger.Adapter)(log)),
		tracer.WithDebugStack(cfg.DebugStack),
		tracer.WithAnalytics(cfg.AnalyticsEnabled),
		tracer.WithSamplerRate(cfg.SamplingRate),
		tracer.WithSamplingRules(lo.Map(cfg.SamplingRules, convertRule)),
	}
	if cfg.MetricsEnabled {
		tracerOpts = append(tracerOpts, tracer.WithRuntimeMetrics())
	}

	if err := tracer.Start(tracerOpts...); err != nil {
		log.Error("Failed to start tracer", logger.Error(err))
	}
	// other observability and telemetry frameworks can be added here if needed

	return func() {
		log.Info("Stopping tracer")
		tracer.Stop()
	}
}
