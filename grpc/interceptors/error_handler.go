package interceptors

import (
	"context"
	"fmt"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryErrorServerInterceptor is a gRPC unary server interceptor that handles errors returned by handlers.
// It detects all errors, tags them in tracing spans, and can be extended for logging and wrapping.
func UnaryErrorServerInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	resp, err := handler(ctx, req)
	err = handleError(ctx, err)
	return resp, err
}

// GrpcErrorStreamingInterceptor is a gRPC streaming server interceptor that handles errors returned by handlers.
// It detects all errors, tags them in tracing spans, and can be extended for logging and wrapping.
func GrpcErrorStreamingInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	_ *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	err := handler(srv, ss)
	err = handleError(ss.Context(), err)
	return err
}

// handleError processes the given error, if any.
func handleError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// Always tag the error in the tracing span for any error
	// ERROR TAG: Error detected and tagged in tracing span for all error types
	setErrorSpan(ctx, err)

	return err
}

// setErrorSpan tags the tracing span with error details if a span exists in the context.
// This includes setting error flags, type, message, and stack trace for observability in tools like Datadog.
// For gRPC status errors, it extracts the code and message specifically.
// For non-status errors, it treats them as system errors.
// ERROR TAG: Tagging error in tracing span with appropriate type, message, and stack
func setErrorSpan(ctx context.Context, err error) {
	fmt.Println("@@@@ Setting error span for:", err)
	span, ok := tracer.SpanFromContext(ctx)
	if !ok {
		return
	}

	span.SetTag(ext.Error, true)

	s, isStatus := status.FromError(err)
	fmt.Println("@@@@ Setting error span for:", s, "isStatus:", isStatus, "message", s.Message(), "code:", s.Code(), s.Code().String())
	if isStatus {
		// For gRPC status errors, use the specific code as error type and the status message
		//span.SetTag(ext.ErrorType, s.Code().String())
		//span.SetTag(ext.ErrorMsg, s.Message())
		// Set the gRPC status code as an integer for visibility in Datadog UI and metrics
		span.SetTag("rpc.grpc.status_code", s.Code())
		span.SetTag("rpc.grpc.status_message", s.Message())
		span.SetTag("some-custom-tag", "custom-value") // Example of adding a custom tag
	} else {
		// For non-gRPC status errors, treat as system error
		span.SetTag(ext.ErrorType, "system")
		span.SetTag(ext.ErrorMsg, err.Error())
	}

	// Set the error stack if available (works with pkg/errors wrapped errors)
	//span.SetTag(ext.ErrorStack, fmt.Sprintf("%+v", err))
}
