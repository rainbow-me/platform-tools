package server

import (
	"net/http"
	"time"

	"google.golang.org/grpc"
)

var (
	DefaultShutdownTimeout   = 30 * time.Second
	DefaultHookTimeout       = 5 * time.Second
	DefaultHTTPReadTimeout   = 5 * time.Second
	DefaultHTTPWriteTimeout  = 10 * time.Second
	DefaultHTTPIdleTimeout   = 120 * time.Second
	DefaultHTTPHeaderTimeout = 2 * time.Second
)

// HTTPConfig holds configuration for HTTP servers
type HTTPConfig struct {
	Name          string        // Unique name for this server (used in logging)
	Address       string        // Address to bind to (e.g., ":8080")
	Handler       http.Handler  // HTTP handler for this server (pre-configured with routes, middlewares, gateways, etc.)
	ReadTimeout   time.Duration // Maximum duration for reading the entire request
	WriteTimeout  time.Duration // Maximum duration before timing out writes
	IdleTimeout   time.Duration // Maximum amount of time to wait for next request when keep-alives are enabled
	HeaderTimeout time.Duration // Amount of time allowed to read request headers
}

// GRPCConfig holds configuration for gRPC servers
type GRPCConfig struct {
	Name       string              // Unique name for this server (used in logging)
	Address    string              // Address to bind to (e.g., ":9090")
	GRPCServer *grpc.Server        // Existing gRPC server instance; if not provided, one will be created
	SetupFunc  func(*grpc.Server)  // Function to register services and configure the gRPC server
	GRPCOpts   []grpc.ServerOption // Server options for creating gRPC server if GRPCServer is nil
}

// HTTPConfigOption is a functional option for configuring HTTPConfig
type HTTPConfigOption func(*HTTPConfig)

// WithHTTPReadTimeout sets the read timeout for the HTTP config
func WithHTTPReadTimeout(timeout time.Duration) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.ReadTimeout = timeout
	}
}

// WithHTTPWriteTimeout sets the write timeout for the HTTP config
func WithHTTPWriteTimeout(timeout time.Duration) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.WriteTimeout = timeout
	}
}

// WithHTTPIdleTimeout sets the idle timeout for the HTTP config
func WithHTTPIdleTimeout(timeout time.Duration) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.IdleTimeout = timeout
	}
}

// WithHTTPHeaderTimeout sets the header timeout for the HTTP config
func WithHTTPHeaderTimeout(timeout time.Duration) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.HeaderTimeout = timeout
	}
}
