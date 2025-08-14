# Correlation

The `correlation` package is designed to manage and propagate correlation data in Go applications, particularly in
distributed systems using gRPC services. Correlation data includes key-value pairs such as correlation IDs, tenancy
information, user IDs, or any custom metadata that needs to be passed across service boundaries for logging, tracing,
and debugging purposes.

Key features:

- **Context-Based Storage**: Stores correlation data in the Go `context.Context` for easy access and propagation.
- **Distributed Tracing Integration**: Automatically sets baggage items on Datadog spans for trace propagation.
- **Logging Support**: Converts correlation data to Zap fields for structured logging.
- **gRPC Propagation**: Uses interceptors to serialize/deserialize correlation data into a single gRPC metadata header (
  `correlation-context` or custom, based on `headers.HeaderXCorrelationID`).
- **Special Helpers**: Built-in support for common keys like `tenancy` and `correlation_id`, with generation of UUID for
  missing correlation IDs.

## Usage

### Initialization

Initialize the Datadog tracer in your `main` function (required for baggage propagation):

```go
import "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"

func main() {
    tracer.Start( /* options, e.g., tracer.WithEnv("prod") */)
    defer tracer.Stop()
    // ... rest of your setup
}
```

### In Handlers (Service Logic)

Use the package to set, get, and manipulate correlation data within your gRPC or HTTP handlers.

#### Setting Data

Set individual keys or bulk values:

```go
ctx = correlation.SetKey(ctx, "user_id", "12345")
ctx = correlation.Set(ctx, map[string]string{"session_id": "abc", "device": "mobile"})

// Special helpers
ctx = correlation.SetTenancy(ctx, "org1")
ctx = correlation.SetID(ctx, uuid.NewString()) // Or auto-generated in interceptor if missing
ctx = correlation.SetIdempotencyKey(ctx, "idempotent_op_123")
```

#### Getting Data

Retrieve values or the entire map:

```go
userID := correlation.GetValue(ctx, "user_id")
tenancy := correlation.Tenancy(ctx)
corrID := correlation.ID(ctx)
ik := correlation.IdempotencyKey(ctx)

allData := correlation.Get(ctx)
hasUser := correlation.Has(ctx, "user_id")
```

#### Modifying Data

Delete or merge:

```go
ctx = correlation.Delete(ctx, "sensitive_key")

otherCtx := // from another source
ctx = correlation.Merge(ctx, otherCtx)
```

#### Logging

Convert to Zap fields:

```go
logger.Info("Request processed", correlation.ToZapFields(ctx)...)
```

This adds fields like `tenancy: "org1"`, `correlation_id: "uuid123"` to logs.

#### Other Utilities

- Check emptiness: `correlation.IsEmpty(ctx)`
- Get keys: `correlation.Keys(ctx)`
- String representation: `correlation.String(ctx)` (e.g., "key1=val1,key2=val2")
- To map: `correlation.ToMap(ctx)`

### In Interceptors (gRPC Propagation)

The package provides unary and stream interceptors to automatically propagate correlation data via gRPC metadata.

#### Server Interceptors

Extract from incoming metadata, parse the `correlation-context` header, set in context, and generate correlation ID if
missing.

Chain before Datadog's tracing interceptor:

```go
import "gopkg.in/DataDog/dd-trace-go.v1/contrib/google.golang.org/grpc" // as grpctrace

grpcServer := grpc.NewServer(
  grpc.ChainUnaryInterceptor(
      correlation.UnaryCorrelationServerInterceptor,
      grpctrace.UnaryServerInterceptor( /* options */),
  )
)
```

#### Client Interceptors

Serialize correlation data from context into outgoing `correlation-context` header.

Chain before Datadog's client interceptor:

```go
conn, err := grpc.Dial("downstream:50051",
  grpc.WithUnaryInterceptor(grpc.ChainUnaryInterceptor(
      correlation.UnaryCorrelationClientInterceptor,
      grpctrace.UnaryClientInterceptor( /* options */),
  ))),
)
```

#### Custom Header

The header name is defined by `ContextCorrelationHeader = headers.HeaderXCorrelationID`. Adjust in your `headers`
package if needed.

### Examples

#### HTTP to gRPC

In a grpc-gateway or custom HTTP setup, extract correlation data from HTTP headers and set it in the context before
calling the downstream gRPC service:

```go
func httpHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Set from HTTP headers
    tenancy := r.Header.Get("Tenancy")
    if tenancy != "" {
    ctx = correlation.SetTenancy(ctx, tenancy)
    }
    
    corrID := r.Header.Get("X-Correlation-Id")
    if corrID != "" {
    ctx = correlation.SetID(ctx, corrID)
    }
    
    // Add custom data from request
    ctx = correlation.SetKey(ctx, "session_id", "abc123")
    
    // Log with correlation fields
    logger.Info("HTTP request received", correlation.ToZapFields(ctx)...)
    
    // Forward to downstream gRPC (interceptors will propagate via metadata)
    grpcResp, err := downstreamClient.YourMethod(ctx, &pb.YourRequest{ /* from r */ })
    if err != nil {
    // Handle error
    }

// Handle response...
}
```

#### gRPC to gRPC

When the gateway serves as a gRPC server (e.g., for internal clients) and forwards to a downstream gRPC service, the
server interceptor automatically extracts and sets correlation data from incoming metadata. In the handler,
access/modify it as needed before calling downstream (client interceptor propagates):

```go
type gatewayService struct {
    pb.UnimplementedYourServiceServer
    downstreamClient pb.YourDownstreamServiceClient // Injected client with interceptors
}

func (s *gatewayService) YourMethod(ctx context.Context, req *pb.YourRequest) (*pb.YourResponse, error) {
    // Correlation data already set by server interceptor from incoming metadata
    logger.Info("gRPC request received", correlation.ToZapFields(ctx)...)
    
    // Optionally modify or add data
    ctx = correlation.SetKey(ctx, "gateway_processed", "true")
    
    // Call downstream gRPC (client interceptor serializes to outgoing metadata)
    downstreamResp, err := s.downstreamClient.DownstreamMethod(ctx, &pb.DownstreamRequest{ /* from req */ })
    if err != nil {
    return nil, err
    }
    
    // Process and return
    return &pb.YourResponse{ /* based on downstreamResp */ }, nil
}

```

#### Downstream gRPC Handler

Access propagated data:

```go
func (s *service) Method(ctx context.Context, req *pb.Req) (*pb.Resp, error) {
    logger.Info("Handling", correlation.ToZapFields(ctx)...)
    
    userID := correlation.GetValue(ctx, "user_id")
    // Use for logic/auth
    
    return resp, nil
}
```

#### Propagation Flow

1. Gateway receives HTTP/GRPC request, sets correlation in ctx.
2. Client interceptor serializes to `correlation-context` header.
3. Downstream server interceptor parses header, sets in ctx.
4. Data available in handlers and traces.