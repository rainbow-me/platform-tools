package server_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rainbow-me/platform-tools/grpc/gateway"
	"github.com/rainbow-me/platform-tools/grpc/protos/gen/go/test"
	"github.com/rainbow-me/platform-tools/grpc/server"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		opts    []server.Option
		wantErr bool
	}{
		{
			name:    "No servers configured",
			opts:    []server.Option{},
			wantErr: false, // No error, but Serve will fail later
		},
		{
			name: "Valid HTTP server",
			opts: []server.Option{
				server.WithHTTPServer("test-http", ":0", http.NewServeMux()),
			},
			wantErr: false,
		},
		{
			name: "Invalid HTTP server no handler",
			opts: []server.Option{
				server.WithHTTPServer("test-http", ":0", nil),
			},
			wantErr: true,
		},
		{
			name: "Valid gRPC server",
			opts: []server.Option{
				server.WithGRPCServer(
					"test-grpc",
					":0",
					nil,
					func(_ *grpc.Server) {},
				),
			},
			wantErr: false,
		},
		{
			name: "Invalid gRPC server no setup",
			opts: []server.Option{
				server.WithGRPCServer(
					"test-grpc",
					":0",
					nil,
					nil,
				),
			},
			wantErr: true,
		},
		{
			name: "Duplicate names",
			opts: []server.Option{
				server.WithHTTPServer("dup", ":0", http.NewServeMux()),
				server.WithGRPCServer(
					"dup",
					":0",
					nil,
					func(_ *grpc.Server) {},
				),
			},
			wantErr: true,
		},
		{
			name: "Duplicate addresses",
			opts: []server.Option{
				server.WithHTTPServer("http1", ":9999", http.NewServeMux()),
				server.WithHTTPServer("http2", ":9999", http.NewServeMux()),
			},
			wantErr: true,
		},

		{
			name: "With shutdown timeout",
			opts: []server.Option{
				server.WithShutdownTimeout(time.Second),
			},
			wantErr: false,
		},
		{
			name: "With shutdown hook",
			opts: []server.Option{
				server.WithShutdownHook(server.ShutdownHook{
					Name:     "test",
					Priority: 1,
					Timeout:  time.Second,
					Hook:     func(_ context.Context) error { return nil },
				},
				),
			},
			wantErr: false,
		},
		{
			name: "With gateway",
			opts: []server.Option{
				server.WithGateway("gateway", ":0",
					nil,
					gateway.WithMux(http.NewServeMux()),
					gateway.WithEndpointRegistration("/test/", test.RegisterHelloServiceHandlerFromEndpoint),
				),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.NewServer(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServer_Serve(t *testing.T) {
	tests := []struct {
		name      string
		opts      []server.Option
		timeout   time.Duration
		expectErr bool
	}{
		{
			name:      "No servers",
			opts:      []server.Option{},
			expectErr: true,
		},
		{
			name: "HTTP server with listener error",
			opts: []server.Option{
				server.WithHTTPServer("test-http", "invalid-address", http.NewServeMux()),
			},
			expectErr: true,
		},
		{
			name: "gRPC server with listener error",
			opts: []server.Option{
				server.WithGRPCServer(
					"test-grpc",
					"invalid-address",
					nil,
					func(_ *grpc.Server) {},
				),
			},
			expectErr: true,
		},
		{
			name: "Successful start and manual stop",
			opts: []server.Option{
				server.WithHTTPServer("test-http", ":0", http.NewServeMux()),
				server.WithSignalHandling(false),
				server.WithAutomaticStop(false),
			},
			timeout:   time.Second,
			expectErr: false, // Manual stop
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := server.NewServer(tt.opts...)
			if err != nil {
				t.Fatalf("NewServer() error = %v", err)
			}

			done := make(chan error)
			go func() {
				done <- srv.Serve()
			}()

			select {
			case err = <-done:
				if (err != nil) != tt.expectErr {
					t.Errorf("Serve() error = %v, expectErr %v", err, tt.expectErr)
				}
			case <-time.After(tt.timeout):
				err = srv.Stop()
				if err != nil {
					return
				}
				<-done // Wait for Serve to exit
			}
		})
	}
}

func TestServer_executeShutdownHooks(t *testing.T) {
	tests := []struct {
		name      string
		opts      []server.Option
		ctx       context.Context
		expectErr bool
	}{
		{
			name:      "No hooks",
			opts:      []server.Option{},
			ctx:       context.Background(),
			expectErr: false,
		},
		{
			name: "Successful hook",
			opts: []server.Option{
				server.WithShutdownHook(server.ShutdownHook{
					Name:     "test",
					Priority: 1,
					Timeout:  time.Second,
					Hook: func(_ context.Context) error {
						return nil
					},
				}),
			},
			ctx:       context.Background(),
			expectErr: false,
		},
		{
			name: "Hook error",
			opts: []server.Option{
				server.WithShutdownHook(server.ShutdownHook{
					Name:     "test",
					Priority: 1,
					Timeout:  time.Second,
					Hook: func(_ context.Context) error {
						return errors.New("hook error")
					},
				}),
			},
			ctx:       context.Background(),
			expectErr: true,
		},
		{
			name: "Hook timeout",
			opts: []server.Option{
				server.WithShutdownHook(server.ShutdownHook{
					Name:     "test",
					Priority: 1,
					Timeout:  time.Second,
					Hook: func(_ context.Context) error {
						time.Sleep(2 * time.Second)
						return nil
					},
				}),
			},
			ctx:       context.Background(),
			expectErr: true,
		},
		{
			name: "Overall shutdown timeout",
			opts: []server.Option{
				server.WithShutdownHook(server.ShutdownHook{
					Name:     "test1",
					Priority: 1,
					Timeout:  time.Second,
					Hook: func(_ context.Context) error {
						time.Sleep(2 * time.Second)
						return nil
					},
				}),
				server.WithShutdownHook(server.ShutdownHook{
					Name:     "test2",
					Priority: 2,
					Timeout:  time.Second,
					Hook: func(_ context.Context) error {
						return nil
					},
				}),
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()
				return ctx
			}(),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := server.NewServer(tt.opts...)
			if err != nil {
				t.Fatalf("NewServer() error = %v", err)
			}
			err = srv.ExecuteShutdownHooks(tt.ctx)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExecuteShutdownHooks() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestServer_Stop(t *testing.T) {
	srv, err := server.NewServer(
		server.WithHTTPServer("test-http", ":0", http.NewServeMux()),
		server.WithAutomaticStop(false),
		server.WithSignalHandling(false),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	done := make(chan error)
	go func() {
		done <- srv.Serve()
	}()

	time.Sleep(100 * time.Millisecond) // Give time to start
	err = srv.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	select {
	case err = <-done:
		if err != nil {
			t.Errorf("Serve() error after Stop = %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Serve did not exit after Stop")
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	srv, err := server.NewServer(
		server.WithHTTPServer("test-http", ":0", http.NewServeMux()),
		server.WithAutomaticStop(false),
		server.WithSignalHandling(false),
		server.WithShutdownTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	done := make(chan error)
	go func() {
		done <- srv.Serve()
	}()

	time.Sleep(100 * time.Millisecond) // Give time to start
	err = srv.GracefulShutdown(context.Background())
	if err != nil {
		t.Errorf("GracefulShutdown() error = %v", err)
	}

	select {
	case err = <-done:
		if err != nil {
			t.Errorf("Serve() error after GracefulShutdown = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Serve did not exit after GracefulShutdown")
	}
}

func TestServer_ShutdownWithHooks(t *testing.T) {
	srv, err := server.NewServer(
		server.WithShutdownHook(server.ShutdownHook{
			Name:     "fast-hook",
			Priority: 1,
			Timeout:  time.Second,
			Hook: func(_ context.Context) error {
				return nil
			},
		}),
		server.WithShutdownHook(server.ShutdownHook{
			Name:     "slow-hook",
			Priority: 2,
			Timeout:  time.Second,
			Hook: func(_ context.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			},
		}),
		server.WithShutdownTimeout(500*time.Millisecond), // Overall timeout short
		server.WithAutomaticStop(false),
		server.WithSignalHandling(false),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	err = srv.GracefulShutdown(context.Background())
	if err == nil || !errors.Is(err, server.ErrShutdownTimeout) {
		t.Errorf("GracefulShutdown() expected ErrShutdownTimeout, got %v", err)
	}
}

// Test signal handling
func TestSignalHandling(t *testing.T) {
	srv, err := server.NewServer(
		server.WithHTTPServer("test-http", ":0", http.NewServeMux()),
		server.WithSignalHandling(true),
		server.WithAutomaticStop(true),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	done := make(chan error)
	go func() {
		done <- srv.Serve()
	}()

	time.Sleep(100 * time.Millisecond) // Give time to start
	err = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	require.NoError(t, err, "Failed to send SIGTERM")

	select {
	case err = <-done:
		if err != nil {
			t.Errorf("Serve() error on signal = %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Serve did not exit on signal")
	}
}

type helloServiceImpl struct {
	test.UnimplementedHelloServiceServer
}

func (s *helloServiceImpl) SayHello(_ context.Context, req *test.SayHelloRequest) (*test.SayHelloResponse, error) {
	parts := strings.Split(req.GetName(), " ")
	firstName := parts[0]
	lastName := ""
	if len(parts) > 1 {
		lastName = parts[1]
	}
	return &test.SayHelloResponse{
		Greeting:  "Hello",
		LastName:  lastName,
		FirstName: firstName,
	}, nil
}

func TestGRPCServerAndClient(t *testing.T) {
	tests := []struct {
		name        string
		reqName     string
		expectGreet string
		expectFirst string
		expectLast  string
		expectErr   bool
	}{
		{
			name:        "Single name",
			reqName:     "John",
			expectGreet: "Hello",
			expectFirst: "John",
			expectLast:  "",
			expectErr:   false,
		},
		{
			name:        "Full name",
			reqName:     "John Doe",
			expectGreet: "Hello",
			expectFirst: "John",
			expectLast:  "Doe",
			expectErr:   false,
		},
		{
			name:        "Empty name",
			reqName:     "",
			expectGreet: "Hello",
			expectFirst: "",
			expectLast:  "",
			expectErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup gRPC server using server package
			grpcPort := ":9050"
			setupFunc := func(s *grpc.Server) {
				test.RegisterHelloServiceServer(s, &helloServiceImpl{})
			}

			srv, err := server.NewServer(
				server.WithGRPCServer("test-grpc", grpcPort, nil, setupFunc),
			)
			if err != nil {
				t.Fatalf("NewServer failed: %v", err)
			}

			done := make(chan error)
			go func() {
				done <- srv.Serve()
			}()

			time.Sleep(time.Second) // Wait for startup

			// gRPC client
			conn, err := grpc.NewClient("localhost"+grpcPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Errorf("gRPC client connection failed: %v", err)
			}
			defer conn.Close()

			client := test.NewHelloServiceClient(conn)
			resp, err := client.SayHello(context.Background(), &test.SayHelloRequest{Name: tt.reqName})
			if (err != nil) != tt.expectErr {
				t.Errorf("SayHello error = %v, expectErr %v", err, tt.expectErr)
			}
			if err == nil {
				if resp.GetGreeting() != tt.expectGreet ||
					resp.GetFirstName() != tt.expectFirst ||
					resp.GetLastName() != tt.expectLast {
					t.Errorf("Unexpected response: got %+v, want greeting=%s, first=%s, last=%s",
						resp, tt.expectGreet, tt.expectFirst, tt.expectLast,
					)
				}
			}

			srv.Stop()
			<-done
		})
	}
}

func TestGRPCGateway(t *testing.T) {
	tests := []struct {
		name        string
		reqName     string
		expectGreet string
		expectFirst string
		expectLast  string
		expectErr   bool
	}{
		{
			name:        "Single name via gateway",
			reqName:     "John",
			expectGreet: "Hello",
			expectFirst: "John",
			expectLast:  "",
			expectErr:   false,
		},
		// Add more cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup gRPC server
			grpcPort := ":9050"
			setupFunc := func(s *grpc.Server) {
				test.RegisterHelloServiceServer(s, &helloServiceImpl{})
			}

			// Setup gateway
			gatewayPort := ":9060"

			srv, err := server.NewServer(
				server.WithGRPCServer("test-grpc", grpcPort, nil, setupFunc),
				server.WithGateway("test-gateway", gatewayPort, nil,
					gateway.WithServerAddress("localhost"+grpcPort),
					gateway.WithEndpointRegistration("/test/", test.RegisterHelloServiceHandlerFromEndpoint),
					gateway.WithCompression(),
					gateway.WithCORS(),
				),
			)
			if err != nil {
				t.Fatalf("NewServer failed: %v", err)
			}

			done := make(chan error)
			go func() {
				done <- srv.Serve()
			}()

			time.Sleep(time.Second) // Wait for startup

			// Test via HTTP (gateway)
			url := "http://localhost" + gatewayPort + "/test/hello"
			// Assume POST with JSON body { "name": tt.reqName }; use http.Client to call
			resp, err := http.Post(url, "application/json", strings.NewReader(fmt.Sprintf(`{"name":"%s"}`, tt.reqName)))
			if (err != nil) != tt.expectErr {
				t.Errorf("Gateway call error = %v, expectErr %v", err, tt.expectErr)
			}
			if err == nil && resp.StatusCode != http.StatusOK {
				t.Errorf("Unexpected status: %d", resp.StatusCode)
			}
			// Parse response JSON and verify (omitted for brevity; add JSON unmarshal and checks)
			fmt.Println(resp.Header)
			srv.Stop()
			<-done
		})
	}
}
