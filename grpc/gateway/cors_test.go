package gateway_test

import (
	"reflect"
	"testing"

	"github.com/rainbow-me/platform-tools/grpc/gateway"
)

func TestCORSOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []gateway.CORSOption
		expected gateway.CORSConfig
	}{
		{
			name:     "No options",
			opts:     nil,
			expected: gateway.CORSConfig{},
		},
		{
			name: "Set origins only",
			opts: []gateway.CORSOption{
				gateway.WithAllowedOrigins([]string{"https://example.com"}),
			},
			expected: gateway.CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
			},
		},
		{
			name: "Set methods only",
			opts: []gateway.CORSOption{
				gateway.WithAllowedMethods([]string{"GET", "POST"}),
			},
			expected: gateway.CORSConfig{
				AllowedMethods: []string{"GET", "POST"},
			},
		},
		{
			name: "Set headers only",
			opts: []gateway.CORSOption{
				gateway.WithAllowedHeaders([]string{"Content-Type"}),
			},
			expected: gateway.CORSConfig{
				AllowedHeaders: []string{"Content-Type"},
			},
		},
		{
			name: "Set credentials true",
			opts: []gateway.CORSOption{
				gateway.WithAllowCredentials(true),
			},
			expected: gateway.CORSConfig{
				AllowCredentials: true,
			},
		},
		{
			name: "Set credentials false",
			opts: []gateway.CORSOption{
				gateway.WithAllowCredentials(false),
			},
			expected: gateway.CORSConfig{
				AllowCredentials: false,
			},
		},
		{
			name: "Combination of all",
			opts: []gateway.CORSOption{
				gateway.WithAllowedOrigins([]string{"https://example.com"}),
				gateway.WithAllowedMethods([]string{"GET", "POST"}),
				gateway.WithAllowedHeaders([]string{"Content-Type"}),
				gateway.WithAllowCredentials(true),
			},
			expected: gateway.CORSConfig{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config gateway.CORSConfig
			for _, opt := range tt.opts {
				opt(&config)
			}
			if !reflect.DeepEqual(config, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, config)
			}
		})
	}
}
