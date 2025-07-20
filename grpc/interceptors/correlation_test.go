package interceptors_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/rainbow-me/platform-tools/grpc/correlation"
	"github.com/rainbow-me/platform-tools/grpc/interceptors"
	pb "github.com/rainbow-me/platform-tools/grpc/protos/gen/go/test"
)

const bufSize = 1024 * 1024

// Mock service implementation for downstream
type echoServer struct {
	pb.UnimplementedEchoServiceServer
}

func (s *echoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	// Return the message and correlation ID from context for test verification
	corrID := correlation.ID(ctx)
	return &pb.EchoResponse{
		Message:       req.GetMessage(),
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
	// Start mock tracer
	err := tracer.Start()
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

			// Set up in-memory listener for downstream server
			listener := bufconn.Listen(bufSize)
			srv := grpc.NewServer(grpc.UnaryInterceptor(interceptors.UnaryCorrelationServerInterceptor))
			pb.RegisterEchoServiceServer(srv, &echoServer{})

			go func() {
				if err = srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
					t.Logf("Server exited with error: %v", err)
				}
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
					t.Errorf("Expected correlation ID '%s', got '%s'", wantID, gotID)
				}
			}
		})
	}
}
