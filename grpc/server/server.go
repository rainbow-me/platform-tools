package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/grpc/gateway"
)

// Option is a functional option for creating a Server
type Option func(*Server) error

// WithLogger sets a custom zap logger for the server
func WithLogger(logger *logger.Logger) Option {
	return func(s *Server) error {
		s.logger = logger
		return nil
	}
}

// WithShutdownTimeout sets the duration for graceful shutdown
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(s *Server) error {
		s.shutdownTimeout = timeout
		return nil
	}
}

// WithHTTPConfig adds an HTTP server configuration
func WithHTTPConfig(config HTTPConfig) Option {
	return func(s *Server) error {
		// Apply defaults
		if config.ReadTimeout == 0 {
			config.ReadTimeout = DefaultHTTPReadTimeout
		}
		if config.WriteTimeout == 0 {
			config.WriteTimeout = DefaultHTTPWriteTimeout
		}
		if config.IdleTimeout == 0 {
			config.IdleTimeout = DefaultHTTPIdleTimeout
		}
		if config.HeaderTimeout == 0 {
			config.HeaderTimeout = DefaultHTTPHeaderTimeout
		}
		if config.Handler == nil {
			return fmt.Errorf("HTTP config %s has no handler", config.Name)
		}
		s.httpConfigs = append(s.httpConfigs, config)
		return nil
	}
}

// WithHTTPServer adds an HTTP server with the given configuration
// The handler should be pre-configured with all routes, middlewares, gateways, health checks, etc.
func WithHTTPServer(name, port string, handler http.Handler, opts ...HTTPConfigOption) Option {
	config := HTTPConfig{
		Name:    name,
		Address: normalizePort(port),
		Handler: handler,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return WithHTTPConfig(config)
}

// WithGRPCConfig adds a gRPC server configuration
func WithGRPCConfig(config GRPCConfig) Option {
	return func(s *Server) error {
		if config.SetupFunc == nil && config.GRPCServer == nil {
			return fmt.Errorf("gRPC config %s has no GRPCServer or SetupFunc", config.Name)
		}
		s.grpcConfigs = append(s.grpcConfigs, config)
		return nil
	}
}

// WithGRPCServer adds a gRPC server with the given configuration
// grpcServer can be an existing server instance; if nil, a new one will be created using grpcOpts.
// The setupFunc is called after server creation (or on the existing one) to register services.
func WithGRPCServer(
	name,
	port string,
	grpcServer *grpc.Server,
	setupFunc func(*grpc.Server),
	grpcOpts ...grpc.ServerOption,
) Option {
	config := GRPCConfig{
		Name:       name,
		Address:    normalizePort(port),
		GRPCServer: grpcServer,
		GRPCOpts:   grpcOpts,
		SetupFunc:  setupFunc,
	}
	return WithGRPCConfig(config)
}

// WithGateway adds a dedicated HTTP server for the gRPC-REST gateway
// It creates an HTTP server with a mux configured solely for the gateway.
func WithGateway(name, port string, httpOpts []HTTPConfigOption, gatewayOpts ...gateway.Option) Option {
	engine := gin.New()
	_, err := gateway.NewGateway(append([]gateway.Option{gateway.WithEngine(engine)}, gatewayOpts...)...)
	if err != nil {
		return func(_ *Server) error {
			return fmt.Errorf("failed to create gateway: %w", err)
		}
	}
	config := HTTPConfig{
		Name:    name,
		Address: normalizePort(port),
		Handler: engine,
	}
	for _, opt := range httpOpts {
		opt(&config)
	}
	return WithHTTPConfig(config)
}

// WithAutomaticStop configures whether the server should automatically stop after the first error
func WithAutomaticStop(isAutomaticStop bool) Option {
	return func(s *Server) error {
		s.isAutomaticStop = isAutomaticStop
		return nil
	}
}

// WithSignalHandling enables or disables signal-based shutdown handling
func WithSignalHandling(enabled bool) Option {
	return func(s *Server) error {
		s.signalHandling = enabled
		return nil
	}
}

// WithShutdownHook adds a shutdown hook to be executed during graceful shutdown
func WithShutdownHook(hook ShutdownHook) Option {
	return func(s *Server) error {
		if hook.Timeout == 0 {
			hook.Timeout = DefaultHookTimeout
		}
		s.shutdownHooks = append(s.shutdownHooks, hook)
		return nil
	}
}

// normalizePort normalizes port to ":port" format
func normalizePort(port string) string {
	if port == "" {
		return ""
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

// Server is a wrapper struct that manages multiple HTTP and gRPC servers with graceful shutdown,
// and shutdown hooks.
type Server struct {
	// Configuration
	shutdownTimeout time.Duration  // Timeout for graceful shutdown
	shutdownHooks   ShutdownHooks  // Cleanup functions to run during shutdown
	httpConfigs     []HTTPConfig   // Configurations for HTTP servers
	grpcConfigs     []GRPCConfig   // Configurations for gRPC servers
	logger          *logger.Logger // Structured logger
	signalHandling  bool           // Whether to handle OS signals
	isAutomaticStop bool           // Whether to auto-stop on first error

	// Runtime state
	httpServers    map[string]*http.Server // Running HTTP servers by name
	grpcServers    map[string]*grpc.Server // Running gRPC servers by name
	serverMu       sync.RWMutex            // Protects httpServers and grpcServers
	shutdownCtx    context.Context         // Context for coordinating shutdown
	shutdownCancel context.CancelFunc      // Function to trigger shutdown
	shutdownOnce   sync.Once               // Ensures shutdown only happens once
	wg             sync.WaitGroup          // Tracks running server goroutines
	errChan        chan error              // Channel for collecting server errors
	signalChan     chan os.Signal          // Channel for OS signals
}

// NewServer creates a Server from the given options.
func NewServer(opts ...Option) (*Server, error) {
	log, err := logger.Instance()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create logger")
	}
	s := &Server{
		shutdownTimeout: DefaultShutdownTimeout,
		httpConfigs:     []HTTPConfig{},
		grpcConfigs:     []GRPCConfig{},
		httpServers:     make(map[string]*http.Server),
		grpcServers:     make(map[string]*grpc.Server),
		isAutomaticStop: true,
		signalHandling:  true,
		logger:          log,
		errChan:         make(chan error, 20),
		signalChan:      make(chan os.Signal, 5),
	}

	s.shutdownCtx, s.shutdownCancel = context.WithCancel(context.Background())

	for _, opt := range opts {
		if err = opt(s); err != nil {
			s.logger.Error("Failed to apply server option", zap.Error(err))
			return nil, err
		}
	}

	// Validate configs
	nameSet := make(map[string]struct{})
	addrSet := make(map[string]struct{})

	for _, config := range s.httpConfigs {
		if _, exists := nameSet[config.Name]; exists {
			return nil, fmt.Errorf("duplicate HTTP server name: %s", config.Name)
		}
		nameSet[config.Name] = struct{}{}
		if config.Address != "" {
			if _, exists := addrSet[config.Address]; exists {
				return nil, fmt.Errorf("duplicate bind address: %s", config.Address)
			}
			addrSet[config.Address] = struct{}{}
		}
		if config.Handler == nil {
			return nil, fmt.Errorf("HTTP server %s has no handler", config.Name)
		}
	}

	for _, config := range s.grpcConfigs {
		if _, exists := nameSet[config.Name]; exists {
			return nil, fmt.Errorf("duplicate gRPC server name: %s", config.Name)
		}
		nameSet[config.Name] = struct{}{}
		if config.Address != "" {
			if _, exists := addrSet[config.Address]; exists {
				return nil, fmt.Errorf("duplicate bind address: %s", config.Address)
			}
			addrSet[config.Address] = struct{}{}
		}
		if config.SetupFunc == nil && config.GRPCServer == nil {
			return nil, fmt.Errorf("gRPC server %s has no GRPCServer or SetupFunc", config.Name)
		}
	}

	s.logger.Info("Server created successfully",
		zap.Duration("shutdown_timeout", s.shutdownTimeout),
		zap.Bool("signal_handling", s.signalHandling),
		zap.Bool("automatic_stop", s.isAutomaticStop),
		zap.Int("http_server_count", len(s.httpConfigs)),
		zap.Int("grpc_server_count", len(s.grpcConfigs)),
		zap.Int("shutdown_hooks", len(s.shutdownHooks)),
	)

	return s, nil
}

// Serve starts all configured servers
func (s *Server) Serve() error {
	if len(s.httpConfigs) == 0 && len(s.grpcConfigs) == 0 {
		return errors.New("no servers configured")
	}

	if s.signalHandling {
		s.setupSignalHandling()
	}

	// Start HTTP servers
	for _, config := range s.httpConfigs {
		s.startHTTPServer(config)
	}

	// Start gRPC servers
	for _, config := range s.grpcConfigs {
		s.startGRPCServer(config)
	}

	s.logger.Info("All servers started")

	var errs []error
	select {
	case sig := <-s.signalChan:
		s.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case err := <-s.errChan:
		errs = append(errs, err)
		for len(s.errChan) > 0 {
			errs = append(errs, <-s.errChan)
		}
		s.logger.Warn("Server error received", zap.Error(err))
	case <-s.shutdownCtx.Done():
		s.logger.Info("Shutdown context cancelled")
	}

	if s.isAutomaticStop {
		if shutdownErr := s.GracefulShutdown(context.Background()); shutdownErr != nil {
			errs = append(errs, shutdownErr)
		}
	}

	s.wg.Wait()
	s.logger.Info("All servers stopped")

	close(s.errChan)
	signal.Stop(s.signalChan) // Stop signal notifications before closing channel
	close(s.signalChan)

	return errors.Join(errs...)
}

// setupSignalHandling configures signal handlers for graceful shutdown
func (s *Server) setupSignalHandling() {
	signal.Notify(s.signalChan, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	go func() {
		for sig := range s.signalChan {
			s.logger.Info("Received signal", zap.String("signal", sig.String()))
			s.shutdownCancel()
			break // Handle first signal only
		}
	}()

	s.logger.Debug("Signal handling configured")
}

// startHTTPServer starts a single HTTP server in a goroutine
func (s *Server) startHTTPServer(config HTTPConfig) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		server := &http.Server{
			Addr:              config.Address,
			Handler:           config.Handler,
			ReadTimeout:       config.ReadTimeout,
			ReadHeaderTimeout: config.HeaderTimeout,
			WriteTimeout:      config.WriteTimeout,
			IdleTimeout:       config.IdleTimeout,
		}

		s.serverMu.Lock()
		s.httpServers[config.Name] = server
		s.serverMu.Unlock()

		s.logger.Info("Starting HTTP server", zap.String("name", config.Name), zap.String("address", config.Address))

		lis, err := net.Listen("tcp", config.Address)
		if err != nil {
			s.logger.Error("Failed to listen", zap.String("name", config.Name), zap.Error(err))
			s.errChan <- fmt.Errorf("HTTP server %s listen error: %w", config.Name, err)
			return
		}

		err = server.Serve(lis)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP server error", zap.String("name", config.Name), zap.Error(err))
			s.errChan <- fmt.Errorf("HTTP server %s error: %w", config.Name, err)
		} else {
			s.logger.Info("HTTP server stopped", zap.String("name", config.Name))
		}

		// Close listener
		if cerr := lis.Close(); cerr != nil && !errors.Is(cerr, net.ErrClosed) {
			s.logger.Warn("Error closing listener", zap.String("name", config.Name), zap.Error(cerr))
		}
	}()
}

// startGRPCServer starts a single gRPC server in a goroutine
func (s *Server) startGRPCServer(config GRPCConfig) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Use existing gRPC server if provided, else create new
		var server *grpc.Server
		if config.GRPCServer != nil {
			server = config.GRPCServer
		} else {
			server = grpc.NewServer(config.GRPCOpts...)
		}

		s.serverMu.Lock()
		s.grpcServers[config.Name] = server
		s.serverMu.Unlock()

		// Register services if setup func provided
		if config.SetupFunc != nil {
			s.logger.Debug("Setting up gRPC services", zap.String("name", config.Name))
			config.SetupFunc(server)
		}

		s.logger.Info("Starting gRPC server", zap.String("name", config.Name), zap.String("address", config.Address))

		lis, err := net.Listen("tcp", config.Address)
		if err != nil {
			s.logger.Error("Failed to listen", zap.String("name", config.Name), zap.Error(err))
			s.errChan <- fmt.Errorf("gRPC server %s listen error: %w", config.Name, err)
			return
		}

		err = server.Serve(lis)
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			s.logger.Error("gRPC server error", zap.String("name", config.Name), zap.Error(err))
			s.errChan <- fmt.Errorf("gRPC server %s error: %w", config.Name, err)
		} else {
			s.logger.Info("gRPC server stopped", zap.String("name", config.Name))
		}

		// Close listener
		if cerr := lis.Close(); cerr != nil && !errors.Is(cerr, net.ErrClosed) {
			s.logger.Warn("Error closing listener", zap.String("name", config.Name), zap.Error(cerr))
		}
	}()
}

// Stop immediately terminates all servers
func (s *Server) Stop() error {
	s.logger.Info("Stopping all servers immediately")
	s.shutdownCancel()
	return s.shutdown(context.Background(), false)
}

// GracefulShutdown performs a graceful shutdown with the configured timeout
func (s *Server) GracefulShutdown(ctx context.Context) error {
	s.logger.Info("Starting graceful shutdown", zap.Duration("timeout", s.shutdownTimeout))

	s.shutdownCancel()

	shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	return s.shutdown(shutdownCtx, true)
}

// shutdown handles the actual shutdown logic
func (s *Server) shutdown(ctx context.Context, isGraceful bool) error { //nolint:gocognit
	var shutdownErr error
	s.shutdownOnce.Do(func() {
		var wg sync.WaitGroup
		errC := make(chan error, len(s.httpServers)+len(s.grpcServers))

		shutdownType := "immediate"
		if isGraceful {
			shutdownType = "graceful"
		}
		s.logger.Info("Shutting down servers", zap.String("type", shutdownType))

		// Shutdown HTTP servers
		s.serverMu.RLock()
		for name, server := range s.httpServers {
			wg.Add(1)
			go func(n string, srv *http.Server) {
				defer wg.Done()
				var err error
				if isGraceful {
					err = srv.Shutdown(ctx)
				} else {
					err = srv.Close()
				}
				if err != nil {
					errC <- fmt.Errorf("HTTP server %s shutdown error: %w", n, err)
				}
			}(name, server)
		}
		s.serverMu.RUnlock()

		// Shutdown gRPC servers
		s.serverMu.RLock()
		for name, server := range s.grpcServers {
			wg.Add(1)
			go func(n string, srv *grpc.Server) {
				defer wg.Done()
				if isGraceful {
					done := make(chan struct{})
					go func() {
						srv.GracefulStop()
						close(done)
					}()
					select {
					case <-done:
					case <-ctx.Done():
						srv.Stop()
						errC <- fmt.Errorf("gRPC server %s graceful shutdown timed out", n)
					}
				} else {
					srv.Stop()
				}
			}(name, server)
		}
		s.serverMu.RUnlock()

		// Wait for completion
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		var errs []error
		select {
		case <-done:
		case <-ctx.Done():
			errs = append(errs, ErrShutdownTimeout)
		}

		close(errC)
		for err := range errC {
			if err != nil {
				errs = append(errs, err)
			}
		}

		shutdownErr = errors.Join(errs...)

		// Execute hooks if graceful
		if isGraceful {
			hookErr := s.ExecuteShutdownHooks(ctx)
			if hookErr != nil {
				shutdownErr = errors.Join(shutdownErr, hookErr)
			}
		}
	})

	return shutdownErr
}

// ExecuteShutdownHooks executes all registered shutdown hooks in priority order
func (s *Server) ExecuteShutdownHooks(ctx context.Context) error {
	if len(s.shutdownHooks) == 0 {
		return nil
	}

	s.logger.Info("Executing shutdown hooks", zap.Int("count", len(s.shutdownHooks)))

	sort.Sort(s.shutdownHooks)

	var wg sync.WaitGroup
	var hookErrs []error
	var mu sync.Mutex // For safe append

	for _, hook := range s.shutdownHooks {
		wg.Add(1)
		go func(h ShutdownHook) {
			defer wg.Done()
			hookCtx, cancel := context.WithTimeout(ctx, h.Timeout)
			defer cancel()

			s.logger.Info("Executing hook", zap.String("name", h.Name), zap.Int("priority", h.Priority))

			start := time.Now()
			err := h.Hook(hookCtx)
			select {
			case <-hookCtx.Done():
				if errors.Is(hookCtx.Err(), context.DeadlineExceeded) {
					mu.Lock()
					hookErrs = append(hookErrs, fmt.Errorf("hook %s timed out", h.Name))
					mu.Unlock()
					s.logger.Error("Hook timed out", zap.String("name", h.Name))
					return // Exit the goroutine early on timeout
				}
			default:
				if err != nil {
					mu.Lock()
					hookErrs = append(hookErrs, err)
					mu.Unlock()
					s.logger.Error("Hook failed", zap.String("name", h.Name), zap.Error(err))
				} else {
					s.logger.Info("Hook completed", zap.String("name", h.Name), zap.Duration("duration", time.Since(start)))
				}
			}
		}(hook)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ErrShutdownTimeout // Overall shutdown timeout for all hooks
	}

	if len(hookErrs) > 0 {
		s.logger.Error("Some shutdown hooks failed", zap.Error(errors.Join(hookErrs...)))
		return errors.Join(hookErrs...)
	}

	s.logger.Info("All shutdown hooks completed")
	return nil
}
