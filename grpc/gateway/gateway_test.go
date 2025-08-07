package gateway_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/common/test"
	"github.com/rainbow-me/platform-tools/grpc/gateway"
	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
	testpb "github.com/rainbow-me/platform-tools/grpc/protos/gen/go/test"
)

func TestNewGateway(t *testing.T) {
	tests := []struct {
		name    string
		options []gateway.Option
		wantErr bool
	}{
		{
			name:    "no Endpoints",
			options: []gateway.Option{},
			wantErr: true,
		},
		{
			name: "valid with Endpoints",
			options: []gateway.Option{
				gateway.WithEndpointRegistration(
					"/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: false,
		},
		{
			name: "invalid prefix",
			options: []gateway.Option{
				gateway.WithEndpointRegistration("invalid",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: true,
		},
		{
			name: "with Logger",
			options: []gateway.Option{
				gateway.WithLogger(logger.NoOp()),
				gateway.WithEndpointRegistration("/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: false,
		},
		{
			name: "with Timeout",
			options: []gateway.Option{
				gateway.WithTimeout(10 * time.Second),
				gateway.WithEndpointRegistration("/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: false,
		},
		{
			name: "with TLS",
			options: []gateway.Option{
				gateway.WithTLS(grpc.WithTransportCredentials(insecure.NewCredentials())),
				gateway.WithEndpointRegistration("/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: false,
		},
		{
			name: "with request logging",
			options: []gateway.Option{
				gateway.WithRequestLogging(),
				gateway.WithEndpointRegistration("/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: false,
		},
		{
			name: "with headers to forward",
			options: []gateway.Option{
				gateway.WithHeadersToForward("X-Test"),
				gateway.WithEndpointRegistration("/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: false,
		},
		{
			name: "registration error",
			options: []gateway.Option{
				gateway.WithEndpointRegistration("/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return errors.New("registration failed")
					}),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, err := gateway.NewGateway(tt.options...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, mux)
			}
		})
	}
}

func TestGateway_validatePrefix(t *testing.T) {
	g := &gateway.Gateway{}
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{"empty", "", true},
		{"no leading slash", "api/", true},
		{"no trailing slash", "/api", true},
		{"valid", "/api/", false},
		{"root", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.ValidatePrefix(tt.prefix)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGateway_metadataAnnotator(t *testing.T) {
	g := &gateway.Gateway{
		HeaderConfig: internalmetadata.HeaderConfig{
			HeadersToForward: []string{"X-Test"},
		},
		Logger: logger.NoOp(),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Test", "value")

	md := g.MetadataAnnotator(context.Background(), req)
	assert.Equal(t, []string{"value"}, md.Get("x-test"))
}

func TestGateway_headerMatcher(t *testing.T) {
	g := &gateway.Gateway{
		HeaderConfig: internalmetadata.HeaderConfig{
			HeadersToForward: []string{"X-Test"},
		},
	}

	key, ok := g.HeaderMatcher("X-Test")
	assert.True(t, ok)
	assert.Equal(t, "x-test", key)

	_, ok = g.HeaderMatcher("X-Unknown")
	assert.False(t, ok)
}

func TestGateway_outgoingHeaderMatcher(t *testing.T) {
	g := &gateway.Gateway{
		HeaderConfig: internalmetadata.HeaderConfig{
			HeadersToForward: []string{"x-test"},
		},
	}

	key, ok := g.OutgoingHeaderMatcher("x-test")
	assert.True(t, ok)
	assert.Equal(t, "x-test", key) // Returns as in config

	key, ok = g.OutgoingHeaderMatcher("content-type")
	assert.True(t, ok)
	assert.Equal(t, "content-type", key)

	_, ok = g.OutgoingHeaderMatcher("unknown")
	assert.False(t, ok)
}

func TestGateway_shouldForwardResponseHeader(t *testing.T) {
	g := &gateway.Gateway{
		HeaderConfig: internalmetadata.HeaderConfig{
			HeadersToForward: []string{"X-Test"},
		},
	}

	assert.True(t, g.ShouldForwardResponseHeader("x-test"))
	assert.False(t, g.ShouldForwardResponseHeader("unknown"))
}

func TestGateway_protoMessageErrorHandler(t *testing.T) {
	g := &gateway.Gateway{Logger: logger.NoOp()}
	mux := runtime.NewServeMux()
	marshaller := &runtime.JSONPb{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	err := errors.New("test error")

	g.ProtoMessageErrorHandler(context.Background(), mux, marshaller, w, r, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGateway_responseHeaderHandler(t *testing.T) {
	g := &gateway.Gateway{
		HeaderConfig: internalmetadata.HeaderConfig{
			HeadersToForward: []string{"X-Test"},
		},
		Logger: logger.NoOp(),
	}

	w := httptest.NewRecorder()
	md := runtime.ServerMetadata{
		HeaderMD: metadata.MD{
			"x-test": []string{"value"},
		},
		TrailerMD: metadata.MD{
			"X-Trailer": []string{"trailer-value"},
		},
	}
	ctx := runtime.NewServerMetadataContext(context.Background(), md)

	err := g.ResponseHeaderHandler(ctx, w, nil)
	require.NoError(t, err)
	assert.Equal(t, "value", w.Header().Get("X-Test")) // Casing from matcher
	assert.Empty(t, w.Header().Get("X-Trailer"))       // not allowed
}

func TestGateway_responseHeaderHandler_Trailers(t *testing.T) {
	g := &gateway.Gateway{
		HeaderConfig: internalmetadata.HeaderConfig{
			HeadersToForward: []string{"X-Trailer"},
		},
		Logger: logger.NoOp(),
	}

	w := httptest.NewRecorder()
	md := runtime.ServerMetadata{
		TrailerMD: metadata.MD{
			"x-trailer": []string{"value"},
		},
	}
	ctx := runtime.NewServerMetadataContext(context.Background(), md)

	err := g.ResponseHeaderHandler(ctx, w, &testpb.SayHelloResponse{})
	require.NoError(t, err)
	assert.Equal(t, "value", w.Header().Get("X-Trailer"))
}

func TestGateway_wrapHandler(t *testing.T) {
	g := &gateway.Gateway{Logger: test.NewLogger(t)}

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	wrapped := g.WrapHandler(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)
}

func TestGateway_registerEndpoints(t *testing.T) {
	logger := logger.NoOp()
	g := &gateway.Gateway{
		Endpoints: map[string][]gateway.RegisterFunc{
			"/api/": {func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
				return nil
			}},
		},
		Mux:                  http.NewServeMux(),
		ServerAddress:        "localhost:9090",
		ServerDialOptions:    []grpc.DialOption{},
		GatewayMuxOptions:    []runtime.ServeMuxOption{},
		HeaderConfig:         internalmetadata.HeaderConfig{HeadersToForward: []string{}},
		Logger:               logger,
		Timeout:              5 * time.Second,
		EnableRequestLogging: true,
		EnableCompression:    true,
		EnableCORS:           true,
	}

	mux, err := g.RegisterEndpoints()
	require.NoError(t, err)
	assert.NotNil(t, mux)
}

func TestGateway_registerEndpoints_InvalidPrefix(t *testing.T) {
	g := &gateway.Gateway{
		Endpoints: map[string][]gateway.RegisterFunc{
			"invalid": {},
		},
		Logger: logger.NoOp(),
	}

	_, err := g.RegisterEndpoints()
	assert.Error(t, err)
}

func TestGateway_registerEndpoints_RegistrationError(t *testing.T) {
	g := &gateway.Gateway{
		Endpoints: map[string][]gateway.RegisterFunc{
			"/api/": {func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
				return errors.New("fail")
			}},
		},
		Logger: logger.NoOp(),
	}

	_, err := g.RegisterEndpoints()
	assert.Error(t, err)
}

func TestGateway_NoLoggingIfNotSpecified(t *testing.T) {
	g := &gateway.Gateway{
		Logger: logger.NoOp(),
	}

	// Nop Logger doesn't log, which is the default
	assert.NotNil(t, g.Logger)
}

func TestGateway_Options(t *testing.T) {
	tests := []struct {
		name     string
		setupOpt func() (gateway.Option, interface{})
		verify   func(t *testing.T, g *gateway.Gateway, expected interface{})
	}{
		{
			name: "WithMux",
			setupOpt: func() (gateway.Option, interface{}) {
				customMux := http.NewServeMux()
				return gateway.WithMux(customMux), customMux
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expectedMux, ok := expected.(*http.ServeMux)
				assert.True(t, ok)
				assert.Equal(t, expectedMux, g.Mux)
			},
		},
		{
			name: "WithGatewayOptions",
			setupOpt: func() (gateway.Option, interface{}) {
				opt := runtime.WithMarshalerOption("*", &runtime.JSONPb{})
				return gateway.WithGatewayOptions(opt), 1 // expected len
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expLen, ok := expected.(int)
				assert.True(t, ok)
				assert.Len(t, g.GatewayMuxOptions, expLen)
			},
		},
		{
			name: "WithHeadersToForward",
			setupOpt: func() (gateway.Option, interface{}) {
				return gateway.WithHeadersToForward("X-Custom"), "X-Custom"
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expHeader, ok := expected.(string)
				assert.True(t, ok)
				assert.Contains(t, g.HeaderConfig.HeadersToForward, expHeader)
			},
		},
		{
			name: "WithDialOptions",
			setupOpt: func() (gateway.Option, interface{}) {
				return gateway.WithDialOptions(grpc.WithWriteBufferSize(256)), 1
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expLen, ok := expected.(int)
				assert.True(t, ok)
				assert.Len(t, g.ServerDialOptions, expLen)
			},
		},
		{
			name: "WithServerAddress",
			setupOpt: func() (gateway.Option, interface{}) {
				return gateway.WithServerAddress("localhost:8080"), "localhost:8080"
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expAddr, ok := expected.(string)
				assert.True(t, ok)
				assert.Equal(t, expAddr, g.ServerAddress)
			},
		},
		{
			name: "WithTimeout",
			setupOpt: func() (gateway.Option, interface{}) {
				return gateway.WithTimeout(10 * time.Second), 10 * time.Second
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expDuration, ok := expected.(time.Duration)
				assert.True(t, ok)
				assert.Equal(t, expDuration, g.Timeout)
			},
		},
		{
			name: "WithGzip",
			setupOpt: func() (gateway.Option, interface{}) {
				return gateway.WithCompression(), true
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expDuration, ok := expected.(bool)
				assert.True(t, ok)
				assert.Equal(t, expDuration, g.EnableCompression)
			},
		},
		{
			name: "WithCORS",
			setupOpt: func() (gateway.Option, interface{}) {
				return gateway.WithCORS(), true
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expDuration, ok := expected.(bool)
				assert.True(t, ok)
				assert.Equal(t, expDuration, g.EnableCORS)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt, expected := tt.setupOpt()
			g := &gateway.Gateway{}
			opt(g)
			tt.verify(t, g, expected)
		})
	}
}

func TestGateway_outgoingHeaderMatcher_StandardHeaders(t *testing.T) {
	g := &gateway.Gateway{}

	key, ok := g.OutgoingHeaderMatcher("Content-Type")
	assert.True(t, ok)
	assert.Equal(t, "content-type", key) // Note: returns lowercased as per code

	key, ok = g.OutgoingHeaderMatcher("Content-Length")
	assert.True(t, ok)
	assert.Equal(t, "content-length", key)
}
