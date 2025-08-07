package gateway

import (
	"net/http"

	"github.com/gorilla/handlers"
)

// CORSOption is a functional option for configuring CORS
type CORSOption func(*CORSConfig)

// WithAllowedOrigins sets the allowed origins for CORS
func WithAllowedOrigins(origins []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedOrigins = origins
	}
}

// WithAllowedMethods sets the allowed methods for CORS
func WithAllowedMethods(methods []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedMethods = methods
	}
}

// WithAllowedHeaders sets the allowed headers for CORS
func WithAllowedHeaders(headers []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedHeaders = headers
	}
}

// WithAllowCredentials sets whether credentials are allowed
func WithAllowCredentials(allow bool) CORSOption {
	return func(c *CORSConfig) {
		c.AllowCredentials = allow
	}
}

// CORS contains all CORS-related configuration and logic
type CORS struct {
	Enabled bool
	Config  CORSConfig
}

// CORSConfig holds the CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// DefaultCORSConfig returns the default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
			http.MethodPatch,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}
}

// Apply wraps the handler with CORS middleware if enabled
func (c *CORS) Apply(handler http.Handler) http.Handler {
	options := []handlers.CORSOption{
		handlers.AllowedOrigins(c.Config.AllowedOrigins),
		handlers.AllowedMethods(c.Config.AllowedMethods),
		handlers.AllowedHeaders(c.Config.AllowedHeaders),
		handlers.OptionStatusCode(http.StatusNoContent),
	}

	if c.Config.AllowCredentials {
		options = append(options, handlers.AllowCredentials())
	}

	return handlers.CORS(options...)(handler)
}
