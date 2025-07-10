package interceptors

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	statusCanceled         = status.New(codes.Canceled, "context canceled")          //nolint:gochecknoglobals
	statusDeadlineExceeded = status.New(codes.DeadlineExceeded, "deadline exceeded") //nolint:gochecknoglobals
)

// contextStatusError wraps a gRPC status with the original context error.
type contextStatusError struct {
	*status.Status
	error
}

// GRPCStatus allows grpc/status.FromError to extract the correct gRPC status code.
func (e *contextStatusError) GRPCStatus() *status.Status {
	return e.Status
}

// Unwrap allows error unwrapping with errors.Is or errors.As.
func (e *contextStatusError) Unwrap() error {
	return e.error
}

// UnaryContextStatusInterceptor maps context-related errors to proper gRPC status codes.
//
// Specifically:
//   - context.Canceled → codes.Canceled
//   - context.DeadlineExceeded → codes.DeadlineExceeded
func UnaryContextStatusInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		switch {
		case errors.Is(err, context.Canceled):
			return resp, &contextStatusError{Status: statusCanceled, error: err}
		case errors.Is(err, context.DeadlineExceeded):
			return resp, &contextStatusError{Status: statusDeadlineExceeded, error: err}
		default:
			return resp, err
		}
	}
}
