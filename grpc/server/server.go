package server

import (
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/rainbow-me/platfomt-tools/grpc/interceptors"
)

// NewServerWithCustomInterceptorChain creates a production-ready gRPC server with a custom
// interceptor chain and sensible defaults. This function abstracts away the complexity of
// setting up interceptors, error handlers, and reflection while providing flexibility
// through additional server options.
//
// Interceptor Chain Processing:
// The interceptor chain is processed in the order they were added, creating a pipeline
// where each interceptor can modify the request/response or perform side effects like
// logging, authentication, or metrics collection.
//
// gRPC Reflection:
// Enables server reflection which allows development tools like grpcurl, grpc_cli,
// and Postman to discover and call available services without requiring .proto files.
// Consider disabling in production environments for security.
//
// Parameters:
//   - chain: Pre-configured interceptor chain containing middleware in execution order
//   - serverOptions: Additional gRPC server options (TLS config, message limits, etc.)
//
// Returns:
//   - *grpc.Server: Fully configured server ready for service registration and startup
//
// Example usage:
//
//	chain := interceptors.NewProductionServerChain(logger, "my-service")
//	server := NewServerWithCustomInterceptorChain(chain,
//	    grpc.MaxRecvMsgSize(4*1024*1024),  // 4MB message limit
//	    grpc.KeepaliveParams(...),         // Connection keepalive
//	)
//
//	pb.RegisterMyServiceServer(server, &serviceImpl{})
//	server.Serve(listener)
func NewServerWithCustomInterceptorChain(
	chain *interceptors.UnaryServerInterceptorChain,
	serverOptions ...grpc.ServerOption,
) *grpc.Server {
	// Transform the interceptor chain into a single chained interceptor function.
	// chain.Commit() finalizes the chain and returns a slice of interceptor functions.
	// grpcmiddleware.ChainUnaryServer() then combines these into a single interceptor
	// that executes them in order, creating a middleware pipeline where each interceptor
	// can wrap the next one in the chain.
	chainedUnaryInterceptor := grpcmiddleware.ChainUnaryServer(chain.Commit())

	// Define the unknown service handler for graceful error handling.
	// This handler is called when clients attempt to invoke methods that haven't been
	// registered with the server. Instead of returning a generic "method not found" error,
	// we return the standard gRPC "Unimplemented" status code which provides clearer
	// semantics and better client-side error handling.
	//
	// Parameters ignored:
	// - srv: The service instance (not needed for error response)
	// - stream: The gRPC server stream (contains request context)
	hndl := func(_ interface{}, _ grpc.ServerStream) error {
		// Return the standard gRPC status for unimplemented methods.
		// This helps clients distinguish between network errors, server errors,
		// and missing method implementations.
		return status.Error(codes.Unimplemented, "Unknown route")
	}

	// Configure the core server options that are essential for this server setup.
	// These options define the fundamental behavior and are applied before any
	// user-provided options, ensuring our base functionality is always present.
	baseServerOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(chainedUnaryInterceptor),
		grpc.UnknownServiceHandler(hndl),
	}

	// Merge base server options with user-provided options.
	// User options are appended after base options, which means they can override
	// base settings if there are conflicts. This gives callers flexibility to
	// customize the server while maintaining essential functionality.
	//
	// Common user options include:
	// - grpc.MaxRecvMsgSize(): Set maximum message size limits
	// - grpc.KeepaliveParams(): Configure connection keepalive behavior
	// - grpc.Creds(): Add TLS/authentication credentials
	// - grpc.MaxConcurrentStreams(): Limit concurrent stream count
	baseServerOptions = append(baseServerOptions, serverOptions...)

	// Create the gRPC server instance with all combined options.
	// At this point, the server has all interceptors configured, error handling
	// set up, and any additional options applied. The server is ready for
	// service registration but not yet started.
	grpcServer := grpc.NewServer(baseServerOptions...)

	// Enable gRPC server reflection for development and debugging.
	// Reflection allows development tools to discover available services and methods
	// without requiring access to .proto files. This is invaluable for:
	// - API testing with tools like grpcurl
	// - Service discovery in development environments
	// - Debugging and introspection
	//
	// Security note: Consider removing this in production environments as it
	// exposes the service interface to anyone who can reach the server.
	reflection.Register(grpcServer)

	return grpcServer
}
