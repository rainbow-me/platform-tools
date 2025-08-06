package interceptors_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rainbow-me/platform-tools/grpc/auth"
	"github.com/rainbow-me/platform-tools/grpc/interceptors"
)

// testHandler is a mock unary handler that returns a fixed response and no error.
// It also records if it was called.
type testHandler struct {
	called bool
}

func (d *testHandler) handle(_ context.Context, _ interface{}) (interface{}, error) {
	d.called = true
	return "success", nil
}

// TestAuthUnaryInterceptor uses table-driven tests to cover all branches and scenarios
// in the AuthUnaryInterceptor and extractToken functions.
func TestAuthUnaryInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *auth.Config
		fullMethod     string
		md             metadata.MD // Metadata to set in context
		expectErr      error       // Expected error (use status.Error for gRPC errors)
		expectCalled   bool        // Whether the handler should be called
		expectResponse interface{} // Expected response if no error
	}{
		{
			name: "Authentication disabled - proceeds to handler",
			cfg: &auth.Config{
				Enabled: false,
			},
			fullMethod:     "/service.Method",
			md:             nil,
			expectErr:      nil,
			expectCalled:   true,
			expectResponse: "success",
		},
		{
			name: "Method in skip list - proceeds to handler",
			cfg: &auth.Config{
				Enabled: true,
				SkipMethods: map[string]bool{
					"/service.SkippedMethod": true,
				},
			},
			fullMethod:     "/service.SkippedMethod",
			md:             nil,
			expectErr:      nil,
			expectCalled:   true,
			expectResponse: "success",
		},
		{
			name: "No metadata in context - returns 'API key not found'",
			cfg: &auth.Config{
				Enabled: true,
			},
			fullMethod:   "/service.Method",
			md:           nil,
			expectErr:    status.Error(codes.Unauthenticated, "API key not found"),
			expectCalled: false,
		},
		{
			name: "Metadata present but no header - returns 'API key not found'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
			},
			fullMethod:   "/service.Method",
			md:           metadata.New(map[string]string{}),
			expectErr:    status.Error(codes.Unauthenticated, "API key not found"),
			expectCalled: false,
		},
		{
			name: "Header present but empty after trim - returns 'API key not found'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "   ",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "API key not found"),
			expectCalled: false,
		},
		{
			name: "Header present but invalid format (no space) - returns 'invalid API key format'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "InvalidToken",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key format"),
			expectCalled: false,
		},
		{
			name: "Header present but scheme mismatch - returns 'invalid API key format'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "Basic token",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key format"),
			expectCalled: false,
		},
		{
			name: "Header present but token empty - returns 'invalid API key format'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "Bearer ",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key format"),
			expectCalled: false,
		},
		{
			name: "Header present but invalid format (multiple parts in token) - returns 'invalid API key format'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "Bearer extra part",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key format"),
			expectCalled: false,
		},
		{
			name: "Valid token but not in keys - returns 'invalid API key provided'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
				Keys: map[string]bool{
					"valid-key": true,
				},
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "Bearer invalid-key",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key provided"),
			expectCalled: false,
		},
		{
			name: "Header with scheme mismatch (no space, wrong prefix) - returns 'invalid API key format'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "TokenBearer 123",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key format"),
			expectCalled: false,
		},
		{
			name: "Scheme mismatch due to extra space in config - returns 'invalid API key format'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer ",
				Keys: map[string]bool{
					"valid-key": true,
				},
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "Bearer valid-key",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key format"),
			expectCalled: false,
		},
		{
			name: "Header with scheme mismatch (no space, wrong prefix) - returns 'invalid API key format'",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "TokenBearer 123",
			}),
			expectErr:    status.Error(codes.Unauthenticated, "invalid API key format"),
			expectCalled: false,
		},

		{
			name: "Valid token in keys - proceeds to handler",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
				Keys: map[string]bool{
					"valid-key": true,
				},
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"authorization": "Bearer  valid-key ", // With spaces to test trim
			}),
			expectErr:      nil,
			expectCalled:   true,
			expectResponse: "success",
		},
		{
			name: "Custom header and scheme - valid",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "X-API-Key",
				Scheme:     "ApiKey",
				Keys: map[string]bool{
					"custom-key": true,
				},
			},
			fullMethod: "/service.Method",
			md: metadata.New(map[string]string{
				"x-api-key": "ApiKey custom-key",
			}),
			expectErr:      nil,
			expectCalled:   true,
			expectResponse: "success",
		},
		{
			name: "Multiple headers - uses first one",
			cfg: &auth.Config{
				Enabled:    true,
				HeaderName: "Authorization",
				Scheme:     "Bearer",
				Keys: map[string]bool{
					"first-key": true,
				},
			},
			fullMethod: "/service.Method",
			md: metadata.MD{
				"authorization": []string{"Bearer first-key", "Bearer second-key"},
			},
			expectErr:      nil,
			expectCalled:   true,
			expectResponse: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with metadata if provided
			ctx := context.Background()
			if tt.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.md)
			}

			// Create the interceptor
			interceptor := interceptors.UnaryAuthUnaryInterceptor(tt.cfg)

			// Mock handler
			dh := &testHandler{}
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return dh.handle(ctx, req)
			}

			// Mock info
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.fullMethod,
			}

			// Call the interceptor
			resp, err := interceptor(ctx, "request", info, handler)

			// Check error
			if !errors.Is(err, tt.expectErr) {
				if statusErr, ok := status.FromError(err); ok {
					expectedStatusErr, _ := status.FromError(tt.expectErr)
					if statusErr.Code() != expectedStatusErr.Code() || statusErr.Message() != expectedStatusErr.Message() {
						t.Errorf("expected error %v, got %v", tt.expectErr, err)
					}
				} else {
					t.Errorf("expected error %v, got %v", tt.expectErr, err)
				}
			}

			// Check if handler was called
			if dh.called != tt.expectCalled {
				t.Errorf("expected handler called: %v, got %v", tt.expectCalled, dh.called)
			}

			// Check response if no error
			if err == nil && resp != tt.expectResponse {
				t.Errorf("expected response %v, got %v", tt.expectResponse, resp)
			}
		})
	}
}
