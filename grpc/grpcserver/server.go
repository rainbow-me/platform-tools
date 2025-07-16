package grpcserver

import (
	"context"
	"time"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/rainbow-me/platform-tools/grpc/interceptors"
)

const (
	// DefaultGRPCMaxMsgSize defines the default gRPC max message size in
	// bytes the server can receive or send.
	DefaultGRPCMaxMsgSize = 1024 * 1024 * 10 // 10MB
)

// NewServerWithCustomInterceptorChain creates a production-ready gRPC server with custom
// interceptor chains and sensible defaults. This function abstracts away the complexity of
// setting up interceptors, error handlers, keepalive, and reflection while providing flexibility
// through additional server options.
//
// Interceptor Chain Processing:
// Unary and stream interceptor chains are processed in the order they were added, creating pipelines
// where each interceptor can modify the request/response or perform side effects like logging,
// authentication, or metrics collection.
//
// gRPC Reflection:
// Optionally enables server reflection, which allows tools like grpcurl to discover services.
// Disabled by default for production security; enable only in development or as needed.
//
// Keepalive Configuration:
// Defaults to conservative keepalive settings to maintain healthy connections and prevent issues
// like idle timeouts or ping floods. These can be overridden via serverOptions.
//
// Parameters:
//   - unaryChain: Pre-configured unary interceptor chain
//   - streamChain: Pre-configured stream interceptor chain (optional; nil to skip)
//   - enableReflection: Whether to enable gRPC reflection (default: false)
//   - serverOptions: Additional gRPC server options (TLS config, custom limits, etc.)
//
// Returns:
//   - *grpc.Server: Fully configured server ready for service registration and startup
//
// Example usage:
//
//	unaryChain := interceptors.NewProductionUnaryChain(logger, "my-service")
//	streamChain := interceptors.NewProductionStreamChain(logger, "my-service")
//	server := NewServerWithCustomInterceptorChain(unaryChain, streamChain, false,
//	    grpc.MaxRecvMsgSize(4*1024*1024),  // 4MB message limit
//	    grpc.KeepaliveParams(...),         // Custom keepalive
//	)
//
//	pb.RegisterMyServiceServer(server, &serviceImpl{})
//	server.Serve(listener)
func NewServerWithCustomInterceptorChain(
	unaryChain *interceptors.UnaryServerInterceptorChain,
	serverOptions ...grpc.ServerOption,
) *grpc.Server {
	// Chain unary interceptors if provided.
	var chainedUnaryInterceptor grpc.UnaryServerInterceptor
	if unaryChain != nil {
		chainedUnaryInterceptor = grpcmiddleware.ChainUnaryServer(unaryChain.Commit())
	} else {
		chainedUnaryInterceptor =
			func(ctx context.Context,
				req interface{},
				info *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler,
			) (interface{}, error) {
				return handler(ctx, req)
			}
	}

	// Unknown service handler for graceful error handling.
	unknownHandler := func(_ interface{}, _ grpc.ServerStream) error {
		return status.Error(codes.Unimplemented, "Unknown route")
	}

	// Base server options with essentials: interceptors, limits, keepalive.
	baseServerOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(chainedUnaryInterceptor),
		grpc.UnknownServiceHandler(unknownHandler),
		grpc.MaxRecvMsgSize(DefaultGRPCMaxMsgSize),
		grpc.MaxSendMsgSize(DefaultGRPCMaxMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second, // Ping every 30s if no activity.
			Timeout: 10 * time.Second, // Wait 10s for ping ack.
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second, // Clients must wait 5s between pings.
			PermitWithoutStream: true,            // Allow pings even without active streams.
		}),
	}

	// Append user-provided options (can override base settings).
	baseServerOptions = append(baseServerOptions, serverOptions...)

	// Create the server.
	grpcServer := grpc.NewServer(baseServerOptions...)

	// Optionally enable reflection.
	reflection.Register(grpcServer)

	return grpcServer
}
