package interceptors_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/rainbow-me/platform-tools/common/correlation"
	"github.com/rainbow-me/platform-tools/common/test"
	"github.com/rainbow-me/platform-tools/grpc/interceptors"
	pb "github.com/rainbow-me/platform-tools/grpc/protos/gen/go/test"
	platformHttp "github.com/rainbow-me/platform-tools/http"
	gininterceptors "github.com/rainbow-me/platform-tools/http/interceptors/gin"
)

// TODO martin add E2E coverage grpc -> resty client -> gin as well

const bufSize = 1024 * 1024

// Request/Response types for Gin server
type Ping struct {
	Message string `json:"message"`
}

// Mock service implementation for downstream
type echoServer struct {
	pb.UnimplementedEchoServiceServer
	ginServerURL string
	client       *resty.Client
}

func (s *echoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	pingReq := Ping{Message: req.GetMessage()}
	var pingResp Ping

	_, err := s.client.R().
		SetContext(ctx).
		SetBody(pingReq).
		SetResult(&pingResp).
		Post(s.ginServerURL + "/ping")

	if err != nil {
		return nil, fmt.Errorf("failed to call gin server: %w", err)
	}

	// Return the message from Gin response and correlation ID from context for test verification
	corrID := correlation.ID(ctx)
	return &pb.EchoResponse{
		Message:       pingResp.Message,
		CorrelationId: corrID,
	}, nil
}

func dialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(_ context.Context, addr string) (net.Conn, error) {
		if addr == "bufnet" {
			return listener.Dial()
		}
		return nil, errors.New("unexpected address: " + addr)
	}
}

func TestCorrelationPropagationE2E(t *testing.T) {
	err := tracer.Start(tracer.WithAgentAddr("localhost:0"), tracer.WithLogger(nil)) // Invalid port to skip connections
	require.NoError(t, err, "Failed to start tracer")
	defer tracer.Stop()

	tests := []struct {
		name              string
		clientCorrData    map[string]string // Correlation data set on client side
		expectGeneratedID bool              // Whether to expect auto-generated ID
	}{
		{
			name:              "propagate existing data with ID",
			clientCorrData:    map[string]string{"tenancy": "org1", "user_id": "123", "correlation_id": "custom-id"},
			expectGeneratedID: false,
		},
		{
			name:              "propagate data without ID - generate on server",
			clientCorrData:    map[string]string{"tenancy": "org1", "user_id": "123"},
			expectGeneratedID: true,
		},
		{
			name:              "no data - generate ID on server",
			clientCorrData:    map[string]string{},
			expectGeneratedID: true,
		},
		{
			name:              "empty values skipped",
			clientCorrData:    map[string]string{"key": "", "valid": "val"},
			expectGeneratedID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var ginCorrelationID string

			// Set up Gin server
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(gininterceptors.DefaultInterceptors(gininterceptors.WithHTTPTrace())...)
			r.POST("/ping", func(c *gin.Context) {
				var req Ping
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ginCorrelationID = correlation.ID(c.Request.Context())
				c.JSON(http.StatusOK, Ping{Message: req.Message})
			})

			ginServer := &http.Server{
				Addr:    "127.0.0.1:0",
				Handler: r,
			}

			ginListener, err := net.Listen("tcp", ginServer.Addr)
			require.NoError(t, err, "Failed to create gin listener")

			ginServerURL := fmt.Sprintf("http://%s", ginListener.Addr().String())

			ginServerDone := make(chan error, 1)
			go func() {
				ginServerDone <- ginServer.Serve(ginListener)
			}()

			defer func() {
				_ = ginServer.Shutdown(ctx)
				select {
				case err := <-ginServerDone:
					if err != nil && !errors.Is(err, http.ErrServerClosed) {
						t.Logf("Gin server stopped with error: %v", err)
					}
				case <-time.After(1 * time.Second):
					t.Log("Timeout waiting for gin server to stop")
				}
			}()

			// Set up in-memory listener for downstream server
			listener := bufconn.Listen(bufSize)
			srv := grpc.NewServer(grpc.UnaryInterceptor(interceptors.UnaryCorrelationServerInterceptor))
			pb.RegisterEchoServiceServer(srv, &echoServer{
				ginServerURL: ginServerURL,
				client:       platformHttp.NewRestyWithClient(http.DefaultClient, test.NewLogger(t)),
			})
			serverDone := make(chan error, 1)
			go func() {
				serverDone <- srv.Serve(listener)
			}()

			defer srv.Stop()

			// Set up client with interceptor
			conn, err := grpc.NewClient( //nolint:govet
				"passthrough:///bufnet",
				grpc.WithContextDialer(dialer(listener)),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithUnaryInterceptor(interceptors.UnaryCorrelationClientInterceptor),
			)
			require.NoError(t, err, "Failed to create client connection")

			defer conn.Close()

			client := pb.NewEchoServiceClient(conn)

			// Prepare client context with correlation data
			clientCtx := correlation.Set(ctx, tt.clientCorrData)

			// Call the method
			resp, err := client.Echo(clientCtx, &pb.EchoRequest{Message: "hello"})
			require.NoError(t, err, "Echo call failed")

			// Verify response message echoed
			if resp.GetMessage() != "hello" {
				t.Errorf("Expected message 'hello', got '%s'", resp.GetMessage())
			}

			// Verify correlation ID
			gotID := resp.GetCorrelationId()
			if tt.expectGeneratedID {
				// Check if it's a valid UUID (generated)
				if _, err = uuid.Parse(gotID); err != nil {
					t.Errorf("Expected generated UUID, got invalid '%s'", gotID)
				}
			} else {
				// Check if matches set ID
				wantID := tt.clientCorrData["correlation_id"]
				if gotID != wantID {
					t.Errorf("Expected correlation ID propagated to gRPC '%s', got '%s'", wantID, gotID)
				}
				if ginCorrelationID != wantID {
					t.Errorf("Expected correlation ID propagated to Gin '%s', got '%s'", wantID, ginCorrelationID)
				}
			}

			// Cleanup: Close client connection first, then gracefully stop server and wait for it to stop
			if err = conn.Close(); err != nil {
				t.Logf("Failed to close conn: %v", err)
			}

			srv.GracefulStop()
			select {
			case err = <-serverDone:
				if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
					t.Errorf("Server stopped with error: %v", err)
				}
			case <-time.After(1 * time.Second):
				t.Error("Timeout waiting for server to stop")
			}
		})
	}
}
