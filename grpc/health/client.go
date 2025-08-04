package health

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// config holds the configuration for creating a health checker.
type config struct {
	target            string
	secure            bool
	backoffConfig     backoff.Config
	minConnectTimeout time.Duration
	dialOptions       []grpc.DialOption
}

// Option is a functional option for configuring the health checker creation.
type Option func(*config)

// WithTarget sets the target address for the gRPC connection (e.g., "localhost:50051").
func WithTarget(target string) Option {
	return func(c *config) {
		c.target = target
	}
}

// WithSecure enables or disables secure (TLS) connection. If false (default),
// uses insecure credentials unless overridden.
// If true, no credentials are set by default; provide custom TLS credentials via WithDialOptions if needed.
func WithSecure(secure bool) Option {
	return func(c *config) {
		c.secure = secure
	}
}

// WithBackoffConfig sets the backoff configuration for connection attempts.
// Uses backoff.Config to avoid deprecated types.
func WithBackoffConfig(bc backoff.Config) Option {
	return func(c *config) {
		c.backoffConfig = bc
	}
}

// WithMinConnectTimeout sets the minimum connect timeout for the connection.
func WithMinConnectTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.minConnectTimeout = timeout
	}
}

// WithDialOptions allows passing custom gRPC DialOptions.
// These are applied last and can override defaults like credentials or connect params.
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(c *config) {
		c.dialOptions = append(c.dialOptions, opts...)
	}
}

// Checker is a wrapper around the gRPC health client that manages the underlying connection.
type Checker struct {
	client grpc_health_v1.HealthClient
	conn   *grpc.ClientConn
}

// Client returns the underlying gRPC health client.
func (c *Checker) Client() grpc_health_v1.HealthClient {
	return c.client
}

// Check performs a health check on the specified service.
func (c *Checker) Check(
	ctx context.Context,
	req *grpc_health_v1.HealthCheckRequest,
	opts ...grpc.CallOption,
) (*grpc_health_v1.HealthCheckResponse, error) {
	return c.client.Check(ctx, req, opts...)
}

// Watch watches the health status of the specified service.
func (c *Checker) Watch(
	ctx context.Context,
	req *grpc_health_v1.HealthCheckRequest,
	opts ...grpc.CallOption,
) (grpc_health_v1.Health_WatchClient, error) {
	return c.client.Watch(ctx, req, opts...)
}

// Close closes the underlying gRPC connection.
func (c *Checker) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// NewHealthChecker creates a new Checker with the provided functional options.
// It creates the connection using the configured parameters.
// The user should call Close() when done, typically with defer.
// Default target is "localhost:50051", insecure, default backoff, and 10s min connect timeout.
func NewHealthChecker(opts ...Option) (*Checker, error) {
	c := &config{
		target:            "localhost:50051",
		secure:            false,
		backoffConfig:     backoff.DefaultConfig,
		minConnectTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.target == "" {
		return nil, errors.New("target address is required")
	}

	var dialOpts []grpc.DialOption
	if !c.secure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	dialOpts = append(dialOpts, grpc.WithConnectParams(grpc.ConnectParams{
		Backoff:           c.backoffConfig,
		MinConnectTimeout: c.minConnectTimeout,
	}))

	dialOpts = append(dialOpts, c.dialOptions...)

	conn, err := grpc.NewClient(c.target, dialOpts...)
	if err != nil {
		return nil, err
	}

	client := grpc_health_v1.NewHealthClient(conn)

	return &Checker{client: client, conn: conn}, nil
}
