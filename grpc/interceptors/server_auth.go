package interceptors

import (
	"context"
	"errors"
	"fmt"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rainbow-me/platform-tools/grpc/auth"
)

// Package-level error definitions for authentication failures.
var (
	errAuthTokenNotFound   = errors.New("auth token not found")
	errInvalidAPIKeyFormat = errors.New("invalid API key format")
)

// UnaryAuthUnaryInterceptor returns a gRPC unary server interceptor that performs API key authentication
// based on the provided configuration. It skips authentication if disabled or for specified methods,
// extracts and validates the API key from metadata, and proceeds to the handler if valid.
func UnaryAuthUnaryInterceptor(cfg *auth.Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authentication if it's disabled in the config.
		if !cfg.Enabled {
			return handler(ctx, req)
		}

		// Skip authentication for specific methods listed in SkipMethods.
		if _, shouldSkip := cfg.SkipMethods[info.FullMethod]; shouldSkip {
			return handler(ctx, req)
		}

		// Extract the API key token from the request metadata.
		token, err := extractToken(ctx, cfg)
		if err != nil {
			// Determine the appropriate error message based on the extraction error.
			var message string
			switch {
			case errors.Is(err, errAuthTokenNotFound):
				message = "API key not found"
			case errors.Is(err, errInvalidAPIKeyFormat):
				message = "invalid API key format"
			default:
				message = "API key validation failed"
			}
			// Return gRPC Unauthenticated status with the error message.
			return nil, status.Error(codes.Unauthenticated, message)
		}

		// Validate if the extracted token matches any of the allowed keys.
		if !cfg.Keys[token] {

			span, ok := tracer.SpanFromContext(ctx)
			fmt.Println("@@@@ Span from context:", span, "ok:", ok)
			//if !ok {
			//	return nil, status.Error(codes.Unauthenticated, "span not found in context")
			//}

			//span.SetTag(ext.Error, true)

			s, isStatus := status.FromError(err)
			fmt.Println("@@@@ Setting error span for:", s, "isStatus:", isStatus)
			if isStatus {
				// For gRPC status errors, use the specific code as error type and the status message
				//span.SetTag(ext.ErrorType, s.Code().String())
				//span.SetTag(ext.ErrorMsg, s.Message())
				// Set the gRPC status code as an integer for visibility in Datadog UI and metrics
				span.SetTag("rpc.grpc.status_code", int(s.Code()))
				span.SetTag("rpc.grpc.status_message", s.Message())
				span.SetTag("some-custom-tag", "custom-value") // Example of adding a custom tag
			} else {
				// For non-gRPC status errors, treat as system error
				//span.SetTag(ext.ErrorType, "system")
				//span.SetTag(ext.ErrorMsg, err.Error())
			}

			return nil, status.Error(codes.Unauthenticated, "invalid API key provided")
		}

		// If authentication succeeds, proceed to the next handler.
		return handler(ctx, req)
	}
}

// extractToken retrieves and parses the authentication token from the gRPC metadata.
// It expects the token in the format "<Scheme> <token>" (e.g., "Bearer xyz") in the specified header,
// where the token itself must not contain spaces and must not be empty.
func extractToken(ctx context.Context, cfg *auth.Config) (string, error) {
	// Retrieve incoming metadata from the context.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errAuthTokenNotFound
	}

	// gRPC metadata keys are always lowercase, so normalize the header name.
	headerKey := strings.ToLower(cfg.HeaderName)
	fullTokenSlice := md.Get(headerKey)
	if len(fullTokenSlice) == 0 {
		return "", errAuthTokenNotFound
	}

	// Take the first value if multiple are present (common case is single value).
	fullToken := strings.TrimSpace(fullTokenSlice[0])
	if fullToken == "" {
		return "", errAuthTokenNotFound
	}

	// Split into exactly two parts: scheme and token.
	parts := strings.SplitN(fullToken, " ", 2)
	if len(parts) != 2 {
		return "", errInvalidAPIKeyFormat
	}

	// Check if the first part matches the expected scheme.
	if parts[0] != cfg.Scheme {
		return "", errInvalidAPIKeyFormat
	}

	// Trim whitespace from the token.
	token := strings.TrimSpace(parts[1])

	// Validate token is not empty and does not contain spaces (as API keys typically don't).
	if token == "" || strings.Contains(token, " ") {
		return "", errInvalidAPIKeyFormat
	}

	return token, nil
}
