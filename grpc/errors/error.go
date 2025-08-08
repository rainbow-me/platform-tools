package errors

import (
	"errors"
	"fmt"

	googleapistatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/rainbow-me/protobuf-registry/schemas/common/gen/v1/go/common"
)

var errInvalidErrorType = errors.New("invalid error type")

type ServiceErrorWrapper struct {
	Detail *common.BackendServiceError
}

type ServiceErrorOption func(*ServiceErrorWrapper)

// WithOriginalError adds the raw error to the BackendServiceError.
func WithOriginalError(err error) ServiceErrorOption {
	return func(detail *ServiceErrorWrapper) {
		if err == nil {
			return
		}
		detail.Detail.Private.RawError = err.Error()
	}
}

// WithMetadata adds metadata to the BackendServiceError.
func WithMetadata(metadata map[string]string) ServiceErrorOption {
	return func(detail *ServiceErrorWrapper) {
		detail.Detail.Private.Metadata = metadata
	}
}

func WithType(t string) ServiceErrorOption {
	return func(detail *ServiceErrorWrapper) {
		detail.Detail.Private.ErrorType = t
	}
}

// WithClientProps adds the Public.CustomMessage to BackendServiceError.
func WithClientProps(code common.InternalErrorCode, message string, details []*anypb.Any) ServiceErrorOption {
	return func(detail *ServiceErrorWrapper) {
		detail.Detail.Public.InternalErrorCode = code
		detail.Detail.Public.CustomMessage = &message
		if details != nil {
			detail.Detail.Public.Details = details
		}
	}
}

// NewServiceError wraps an error in a gRPC status, with details set to a common.BackendServiceError.
//
// Publicly exposed:
// - code: The gRPC code corresponding to the status (e.g. codes.NotFound).
// - internalErrCode: The common_api global error.
//
// Exposed only internally (for debugging):
// - message: The internal message used for debugging, not forwarded to client.
func NewServiceError(code codes.Code, message string, opts ...ServiceErrorOption) error {
	backendErr := &ServiceErrorWrapper{
		Detail: &common.BackendServiceError{
			Public: &common.BackendServiceError_Public{
				InternalErrorCode: common.InternalErrorCode_INTERNAL_ERROR_CODE_UNSPECIFIED,
				CustomMessage:     nil, // Can be populated using WithCustomMessage
			},
			Private: &common.BackendServiceError_Private{
				Message:   message,
				ErrorType: "",  // Can be populated using WithType
				RawError:  "",  // Can be populated using WithOriginalError
				Metadata:  nil, // Can be populated using WithMetadata
			},
		},
	}

	for i := range opts {
		opts[i](backendErr)
	}

	bytesVal, marshalErr := proto.Marshal(backendErr.Detail)
	if marshalErr != nil {
		return status.Errorf(codes.Internal, "failed to marshal BackendServiceError: %v", marshalErr)
	}

	err := status.Newf(code, backendErr.Detail.GetPrivate().GetMessage())
	errProto := err.Proto()
	errProto.Details = append(errProto.GetDetails(), &anypb.Any{Value: bytesVal})

	return status.ErrorProto(errProto)
}

// extractBackendError parses a gRPC status and attempts to extract a BackendServiceError from its details.
func extractBackendError(statusProto *googleapistatus.Status) (*common.BackendServiceError, error) {
	details := statusProto.GetDetails()
	if len(details) == 0 {
		return nil, errInvalidErrorType
	}

	for _, detail := range details {
		var backendErr common.BackendServiceError
		if err := proto.Unmarshal(detail.GetValue(), &backendErr); err != nil {
			continue // Ignore malformed entries and try the next
		}
		return &backendErr, nil
	}

	return nil, errInvalidErrorType
}

// ParseBackendServiceError extracts the custom BackendServiceError from a generic gRPC error.
// Returns nil if the error does not contain a valid BackendServiceError detail.
func ParseBackendServiceError(err error) (*common.BackendServiceError, error) {
	st := status.Convert(err)
	if st == nil {
		return nil, fmt.Errorf("failed to convert error to gRPC status: %w", err)
	}

	return extractBackendError(st.Proto())
}
