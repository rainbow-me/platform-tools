# gRPC REST Gateway

This package provides a configurable REST gateway for gRPC services using the `grpc-ecosystem/grpc-gateway/v2` library.
It allows you to expose gRPC APIs over HTTP/JSON with customizable options like endpoint prefixes, header forwarding,
logging, and more. Ideal for building scalable backend services where gRPC is the internal protocol, but REST is needed
for external clients.

## Features

- **Functional Options Pattern**: Easily configure the gateway with options for server address, dial options, TLS,
  timeouts, custom loggers, and more.
- **Endpoint Prefixing**: Register gRPC services under custom URL prefixes (e.g., `/api/v1/`) to support versioning and
  namespacing.
- **Header Propagation**: Control which HTTP headers are forwarded to gRPC metadata and vice versa.
- **Custom Error and Response Handling**: Built-in handlers for gRPC errors and metadata in HTTP responses.
- **Shared gRPC Connection**: Efficiently shares a single gRPC connection across registrations (in optimized versions).
- **Validation and Defaults**: Sensible defaults with prefix validation to prevent common misconfigurations.

## Usage

### Basic Setup

1. Generate gRPC and Gateway code from your `.proto` files using `protoc` with the `grpc-gateway` plugin.
2. Import the package and create the gateway:

```go
package main

import (
  "net/http"

  "github.com/yourusername/backend-service-template/internal/gateway"
  // Import your generated pb package
  "yourpbpackage"
)

func main() {
  mux, err := gateway.NewGateway(
    gateway.WithServerAddress("localhost:9090"),
    gateway.WithEndpointRegistration("/api/v1/", yourpbpackage.RegisterYourServiceHandler),
    gateway.WithHeadersToForward("Authorization", "X-Request-ID"),
  )
  if err != nil {
    panic(err)
  }

  http.ListenAndServe(":8080", mux)
}
```

- The gateway connects to your gRPC server at the specified address and exposes REST endpoints under `/api/v1/`.

### Advanced Configuration

Use functional options to customize:

- **WithTLS**: Enable TLS for gRPC connections.
- **WithTimeout**: Set dial timeout (default: 30s).
- **WithLogger**: Provide a custom `zap.Logger`.
- **WithMux**: Use an existing `http.ServeMux`.
- **WithGatewayOptions**: Add extra `runtime.ServeMuxOption` for advanced grpc-gateway config.

## Examples

### Register Multiple Services

```go
s, err := gateway.NewGateway(
  gateway.WithEndpointRegistration("/user/v1/", pb.RegisterUserServiceHandler),
  gateway.WithEndpointRegistration("/payment/v1/", pb.RegisterPaymentServiceHandler),
)
```

- User endpoints at `/user/v1/...`, payments at `/payment/v1/...`.