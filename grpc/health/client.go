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
	target      string
	dialTimeout time.Duration
	dialOptions []grpc.DialOption
}

// Option is a functional option for configuring the health checker creation.
type Option func(*config)

// WithTarget sets the target address for the gRPC connection (e.g., "localhost:50051").
func WithTarget(target string) Option {
	return func(c *config) {
		c.target = target
	}
}

// WithDialOptions allows passing custom gRPC DialOptions.
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(c *config) {
		c.dialOptions = append(c.dialOptions, opts...)
	}
}

// HealthChecker is a wrapper around the gRPC health client that manages the underlying connection.
type HealthChecker struct {
	client grpc_health_v1.HealthClient
	conn   *grpc.ClientConn
}

// Check performs a health check on the specified service.
func (h *HealthChecker) Check(
	ctx context.Context,
	req *grpc_health_v1.HealthCheckRequest,
	opts ...grpc.CallOption,
) (*grpc_health_v1.HealthCheckResponse, error) {
	return h.client.Check(ctx, req, opts...)
}

// Watch watches the health status of the specified service.
func (h *HealthChecker) Watch(
	ctx context.Context,
	req *grpc_health_v1.HealthCheckRequest,
	opts ...grpc.CallOption,
) (grpc_health_v1.Health_WatchClient, error) {
	return h.client.Watch(ctx, req, opts...)
}

// Close closes the underlying gRPC connection.
func (h *HealthChecker) Close() error {
	if h.conn != nil {
		return h.conn.Close()
	}
	return nil
}

// NewHealthChecker creates a new HealthChecker with the provided functional options.
// It creates the connection and waits for it to be ready if a dial timeout is set.
// The user should call Close() when done, typically with defer.
// Default target is "localhost:50051", insecure, and 10s dial timeout.
func NewHealthChecker(opts ...Option) (*HealthChecker, error) {
	c := &config{
		target:      "localhost:50051",
		dialTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.target == "" {
		return nil, errors.New("target address is required")
	}

	dialOpts := c.dialOptions

	// set insecure credentials if none provided
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// Set connect parameters with the dial timeout
	connectParams := grpc.ConnectParams{
		Backoff:           backoff.DefaultConfig,
		MinConnectTimeout: c.dialTimeout,
	}
	dialOpts = append(dialOpts, grpc.WithConnectParams(connectParams))

	conn, err := grpc.NewClient(c.target, dialOpts...)
	if err != nil {
		return nil, err
	}

	client := grpc_health_v1.NewHealthClient(conn)

	return &HealthChecker{client: client, conn: conn}, nil
}
