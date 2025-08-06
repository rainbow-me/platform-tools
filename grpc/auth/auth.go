package auth

const (
	DefaultHeaderName = "Authorization"
	DefaultScheme     = "Bearer"
)

// ConfigOption is a functional option for configuring the interceptor chain
type ConfigOption func(*Config)

// WithAuthHeaderName sets the header name for authentication
func WithAuthHeaderName(name string) ConfigOption {
	return func(c *Config) {
		c.HeaderName = name
	}
}

// WithAuthScheme sets the scheme for authentication (e.g., "Bearer")
func WithAuthScheme(scheme string) ConfigOption {
	return func(c *Config) {
		c.Scheme = scheme
	}
}

// WithSkipAuthMethods adds methods to skip authentication verification for
func WithSkipAuthMethods(methods ...string) ConfigOption {
	return func(c *Config) {
		for _, method := range methods {
			c.SkipMethods[method] = true
		}
	}
}

// WithSimpleAuth enables simple API key authentication
func WithSimpleAuth(isEnable bool, keys ...string) ConfigOption {
	return func(c *Config) {
		c.Enabled = isEnable

		for _, method := range keys {
			c.Keys[method] = true
		}
	}
}

// Config holds the authentication configuration settings
type Config struct {
	Enabled     bool
	HeaderName  string
	Scheme      string
	Keys        map[string]bool // For static API key verification; supports multiple keys
	SkipMethods map[string]bool // Methods to skip authentication verification for
}
