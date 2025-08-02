package gateway_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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
			name: "with middleware ",
			options: []gateway.Option{
				gateway.WithGinMiddlewares(),
				gateway.WithEndpointRegistration("/api/",
					func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
						return nil
					}),
			},
			wantErr: false,
		},

		{
			name: "with http handlers",
			options: []gateway.Option{
				gateway.WithHTTPHandlers(),
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

func TestGateway_registerEndpoints(t *testing.T) {
	// Dummy generated register function simulation
	dummyGeneratedRegister := func(
		_ context.Context,
		mux *runtime.ServeMux,
		_ string,
		_ []grpc.DialOption,
	) error {
		// Simulate registering a handler as in generated code
		// Here, we manually add a test handler to mimic a proxied endpoint
		err := mux.HandlePath(
			"GET",
			"/test",
			func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
				_, _ = w.Write([]byte(r.URL.Path))
			})
		if err != nil {
			return err
		}
		// In real generated code, this would register multiple handlers via RegisterHandlerFromEndpoint or similar,
		// but for test, we just add one dummy handler
		return nil
	}

	g := &gateway.Gateway{
		Endpoints: map[string][]gateway.RegisterFunc{
			"/api/": {dummyGeneratedRegister},
		},
		Engine:            gin.New(),
		ServerAddress:     "localhost:9090",
		ServerDialOptions: []grpc.DialOption{},
		GatewayMuxOptions: []runtime.ServeMuxOption{},
		HeaderConfig:      internalmetadata.HeaderConfig{HeadersToForward: []string{}},
		Logger:            logger.NoOp(),
		HealthServer:      health.NewServer(),
		HealthEndpoint:    "/health",
		Timeout:           5 * time.Second,
	}

	engine, err := g.RegisterEndpoints()
	require.NoError(t, err)
	assert.NotNil(t, engine)

	// Test prefix stripping
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/test", w.Body.String())
}

func TestGateway_registerEndpoints_1(t *testing.T) {
	// Dummy generated register function simulation
	dummyGeneratedRegister := func(
		_ context.Context,
		mux *runtime.ServeMux,
		_ string,
		_ []grpc.DialOption,
	) error {
		// Simulate registering a handler as in generated code
		// Here, we manually add a test handler to mimic a proxied endpoint
		err := mux.HandlePath(
			http.MethodGet,
			"/test",
			func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
				raw := r.URL.RawPath
				if raw == "" {
					raw = "empty"
				}
				_, _ = w.Write([]byte(r.URL.Path + "|" + raw))
			})
		if err != nil {
			return err
		}
		// In real generated code, this would register multiple handlers via RegisterHandlerFromEndpoint or similar,
		// but for test, we just add one dummy handler
		return nil
	}

	g := &gateway.Gateway{
		Endpoints: map[string][]gateway.RegisterFunc{
			"/api/": {dummyGeneratedRegister},
		},
		Engine:            gin.New(),
		ServerAddress:     "localhost:9090",
		ServerDialOptions: []grpc.DialOption{},
		GatewayMuxOptions: []runtime.ServeMuxOption{},
		HeaderConfig:      internalmetadata.HeaderConfig{HeadersToForward: []string{}},
		Logger:            logger.NoOp(),
		HealthServer:      health.NewServer(),
		HealthEndpoint:    "/health",
		Timeout:           5 * time.Second,
	}

	mux, err := g.RegisterEndpoints()
	require.NoError(t, err)
	assert.NotNil(t, mux)

	tests := []struct {
		name         string
		path         string
		rawPath      string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "basic path stripping without rawpath",
			path:         "/api/test",
			rawPath:      "",
			expectedCode: http.StatusOK,
			expectedBody: "/test|empty",
		},
		{
			name:         "path and rawpath stripping with encoding",
			path:         "/api/te%73t",
			rawPath:      "/api/te%73t",
			expectedCode: http.StatusOK,
			expectedBody: "/test|/te%73t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.URL.RawPath = tt.rawPath
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestGateway_CustomRegistrars(t *testing.T) {
	dummyRegister := func(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
		return nil
	}

	customRegistrar := func(engine *gin.Engine) {
		engine.GET("/custom", func(c *gin.Context) {
			c.String(http.StatusOK, "custom response")
		})
	}

	engine, err := gateway.NewGateway(
		gateway.WithEndpointRegistration("/api/", dummyRegister),
		gateway.WithHTTPHandlers(customRegistrar),
	)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/custom", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "custom response", w.Body.String())
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
			name: "WithGinEngine",
			setupOpt: func() (gateway.Option, interface{}) {
				customMux := gin.New()
				return gateway.WithEngine(customMux), customMux
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				expectedMux, ok := expected.(*gin.Engine)
				assert.True(t, ok)
				assert.Equal(t, expectedMux, g.Engine)
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
			name: "WithHealthCheck",
			setupOpt: func() (gateway.Option, interface{}) {
				hs := health.NewServer()
				endpoint := "/health"
				return gateway.WithHealthCheck(hs, endpoint), struct {
					HS       *health.Server
					Endpoint string
				}{HS: hs, Endpoint: endpoint}
			},
			verify: func(t *testing.T, g *gateway.Gateway, expected interface{}) {
				exp, ok := expected.(struct {
					HS       *health.Server
					Endpoint string
				})
				assert.True(t, ok)
				assert.Equal(t, exp.HS, g.HealthServer)
				assert.Equal(t, exp.Endpoint, g.HealthEndpoint)
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

// faultyRecorder to force encode error
type faultyRecorder struct {
	*httptest.ResponseRecorder
}

func (f *faultyRecorder) Write([]byte) (int, error) {
	return 0, errors.New("encode failed")
}

func TestGateway_healthHandler(t *testing.T) {
	tests := []struct {
		name           string
		service        string
		setupStatus    map[string]grpc_health_v1.HealthCheckResponse_ServingStatus // Map service to status
		expectedCode   int
		expectedBody   string
		expectedHeader string
		forceEncodeErr bool
		expectErrLog   bool
	}{
		{
			name:    "successful check",
			service: "test-service",
			setupStatus: map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
				"test-service": grpc_health_v1.HealthCheckResponse_SERVING,
			},
			expectedCode:   http.StatusOK,
			expectedBody:   `{"status":"SERVING"}`,
			expectedHeader: "application/json; charset=utf-8",
		},
		{
			name:    "not serving",
			service: "test-service",
			setupStatus: map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
				"test-service": grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			},
			expectedCode:   http.StatusOK,
			expectedBody:   `{"status":"NOT_SERVING"}`,
			expectedHeader: "application/json; charset=utf-8",
		},
		{
			name:           "unknown service error",
			service:        "unknown",
			setupStatus:    map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{}, // No status set
			expectedCode:   http.StatusServiceUnavailable,
			expectedBody:   `{"error":"rpc error: code = NotFound desc = unknown service"}`,
			expectedHeader: "application/json; charset=utf-8",
			expectErrLog:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hs := health.NewServer()
			for svc, status := range tt.setupStatus {
				hs.SetServingStatus(svc, status)
			}
			g := &gateway.Gateway{
				HealthServer: hs,
				Logger:       test.NewLogger(t),
			}

			handler := g.HealthHandler()

			req := httptest.NewRequest(http.MethodGet, "/health?service="+tt.service, nil)
			w := httptest.NewRecorder()
			if tt.forceEncodeErr {
				fw := &faultyRecorder{ResponseRecorder: httptest.NewRecorder()}
				w = fw.ResponseRecorder // Use the underlying recorder for assertions
				c, _ := gin.CreateTestContext(fw)
				c.Request = req
				handler(c)
			} else {
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				handler(c)
			}

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Equal(t, tt.expectedBody, strings.TrimSpace(w.Body.String()))
			assert.Equal(t, tt.expectedHeader, w.Header().Get("Content-Type"))
		})
	}
}
