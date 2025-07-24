# Server

## Overview

The `server` package is a Go library designed to simplify the management of multiple HTTP and gRPC servers in a single
application. It provides a unified interface for starting, stopping, and gracefully shutting down servers, with support
for configurable timeouts, shutdown hooks, signal handling, and automatic error-based shutdown. This package is ideal
for backend services that need to run multiple APIs (REST via HTTP and RPC via gRPC) concurrently while ensuring clean
resource cleanup during shutdown.

## Features

- **Multi-Server Support**: Run multiple HTTP and gRPC servers in parallel.
- **Graceful Shutdown**: Supports timeouts for shutdown and individual hooks.
- **Shutdown Hooks**: Register prioritized functions for cleanup during shutdown, with per-hook timeouts.
- **Signal Handling**: Optionally listen for OS signals (e.g., Ctrl+C, SIGTERM) to trigger shutdown.
- **Automatic Stop**: Optionally shut down all servers on the first error.
- **Configurable Timeouts**: Customize read/write/idle/header timeouts for HTTP servers.
- **gRPC-REST Gateway Support**: Easily add a dedicated HTTP server for gRPC gateway.
- **Logging**: Integrated with zap for structured logging.

## Usage

### Basic Setup

Create a new server instance using functional options. Start with `server.NewServer(...)` and add servers/hooks as
needed.

```go
package main

import (
	"context"
	"fmt"
	"github.com/rainbow-me/backend-service-template/internal/pkg/grpcserver"
	"github.com/rainbow-me/backend-service-template/internal/pkg/interceptors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rainbow-me/backend-service-template/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, _ := zap.NewProduction() // Handle error in production

	chain := interceptors.NewDefaultServerUnaryChain(
		"test-service",
		"development",
		logger,
		interceptors.WithBasicLogging(true, zap.DebugLevel),
	)
	grpcServer := grpcserver.NewServerWithCustomInterceptorChain(chain)

	srv, err := server.NewServer(
		server.WithLogger(logger),
		server.WithShutdownTimeout(15*time.Second), // Custom shutdown timeout
		server.WithSignalHandling(true),            // Enable OS signal handling
		server.WithAutomaticStop(true),             // Auto-shutdown on errors

		// Add HTTP server (port as string, e.g., "8080" or ":8080")
		server.WithHTTPServer("api-http", "8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, HTTP!")
		}),
			server.WithHTTPReadTimeout(10*time.Second),
		),

		// Add gRPC server
		server.WithGRPCServer("api-grpc", "9090", grpcServer, func(s *grpc.Server) {
			// Register gRPC services here
		}),

		// Add shutdown hook
		server.WithShutdownHook(server.ShutdownHook{
			Name:     "db-close",
			Priority: 1,
			Timeout:  5 * time.Second,
			Hook: func(ctx context.Context) error {
				log.Println("Closing database...")
				return nil
			},
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Serve(); err != nil {
		log.Printf("Server exited with error: %v", err)
		os.Exit(1)
	}
}
```

### Adding a gRPC-REST Gateway

Use `WithGateway` to add a dedicated HTTP server for gRPC gateway:

```go
package main

import (
	"github.com/rainbow-me/backend-service-template/internal/pkg/gateway"
	"github.com/rainbow-me/backend-service-template/internal/pkg/grpcserver"
	"github.com/rainbow-me/backend-service-template/internal/pkg/interceptors"
	"github.com/rainbow-me/backend-service-template/internal/pkg/server"
	test "github.com/rainbow-me/backend-service-template/protos/gen/go/v1"
	"google.golang.org/grpc"
	"os"
)

func main() {
	chain := interceptors.NewDefaultServerUnaryChain(
		"test-service",
		"development",
		logger,
		interceptors.WithBasicLogging(true, zap.DebugLevel),
	)

	grpcServer := grpcserver.NewServerWithCustomInterceptorChain(chain)

	srv, err := server.NewServer(
		server.WithGRPCServer(
			"api-grpc",
			":9090",
			grpcServer,
			func(s *grpc.Server) {
				test.RegisterHelloServiceServer(s, &HelloServerImpl{})
			},
		),

		server.WithGateway(
			"gateway-http",
			":8082",
			nil,
			gateway.WithEndpointRegistration("/test/", test.RegisterHelloServiceHandlerFromEndpoint), // assumes proto
			gateway.WithServerAddress("localhost:9090"),
		),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Serve(); err != nil {
		log.Printf("Server exited with error: %v", err)
		os.Exit(1)
	}
}

```

### Graceful Shutdown

- Call `srv.GracefulShutdown(ctx)` manually or let signals/errors trigger it (if enabled).
- Shutdown hooks run in priority order, each with its own timeout.
- Overall shutdown respects `shutdownTimeout`; if exceeded, returns `ErrShutdownTimeout`.

### Immediate Stop

- Call `srv.Stop()` to terminate servers without grace period (no hooks run).

### Configuration Options

- **WithLogger(logger logger.Logger)**: Set custom logger.
- **WithShutdownTimeout(d time.Duration)**: Set graceful shutdown timeout.
- **WithHTTPServer(name, port string, handler http.Handler, opts ...HTTPConfigOption)**: Add HTTP server.
    - Options: `WithHTTPReadTimeout`, `WithHTTPWriteTimeout`, etc.
- **WithGRPCServer(name, port string, grpcServer *grpc.Server, setupFunc func(
  *grpc.Server), grpcOpts ...grpc.ServerOption, )**: Add gRPC server.
- **WithGateway(name, port string, gatewayOpts []gateway.Option, httpOpts ...HTTPConfigOption)**: Add gRPC gateway as
  HTTP server.
- **WithAutomaticStop(bool)**: Enable/disable auto-shutdown on errors (default: true).
- **WithSignalHandling(bool)**: Enable/disable OS signal listening (default: true).
- **WithShutdownHook(hook ShutdownHook)**: Add cleanup hook.

### Error Handling

- Server creation validates duplicates (names/ports) and required fields.
- Runtime errors (e.g., bind failures) are sent to `errChan` and can trigger shutdown.
- Shutdown errors (e.g., timeouts) are aggregated and returned.