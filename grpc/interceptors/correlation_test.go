package interceptors_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/rainbow-me/platform-tools/common/env"
	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/grpc/grpcserver"
	"net"
	"os"
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

	corrID := correlation.ID(ctx)
	// Return the message and correlation ID from context for test verification
	info, ok := interceptors.GetRequestInfoFromContext(ctx)
	fmt.Println("Request Info:", info, ok, "Correlation ID:", corrID)
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
			l, _ := logger.Instance()
			// Set up in-memory listener for downstream server
			listener := bufconn.Listen(bufSize)
			//srv := grpc.NewServer(
			//	//grpc.UnaryInterceptor(interceptors.UnaryCorrelationServerInterceptor)
			//	grpc.ChainUnaryInterceptor(
			//		interceptors.RequestContextUnaryServerInterceptor(),
			//		interceptors.UnaryCorrelationServerInterceptor,
			//	),
			//)

			// Set up the interceptor chain for gRPC requests.
			chain := interceptors.NewDefaultServerUnaryChain(
				"test-service",
				os.Getenv(env.ApplicationEnvKey),
				l,
				interceptors.WithBasicLogging(true, true, logger.DebugLevel),
			)
			srv := grpcserver.NewServerWithCustomInterceptorChain(chain)
			pb.RegisterEchoServiceServer(srv, &echoServer{})
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
				grpc.WithChainUnaryInterceptor(
					interceptors.UnaryRequestContextClientInterceptor,
					interceptors.UnaryCorrelationClientInterceptor,
					interceptors.UnaryUpstreamInfoClientInterceptor("test-service"),
				),
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
