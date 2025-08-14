package headers

// Request Identification Headers
const (
	// HeaderXRequestID is used to uniquely identify individual HTTP requests
	// for logging, debugging, and tracking purposes across the application
	HeaderXRequestID = "x-request-id"
)

// Correlation and Trace ID Headers
const (
	// HeaderXCorrelationID is used to correlate related requests across multiple
	// services in a distributed system, allowing you to track the complete flow
	// of a business transaction through various microservices
	HeaderXCorrelationID = "x-correlation-id"

	// HeaderXTraceID is used for distributed tracing to track requests across
	// multiple services and create a complete trace of the request journey.
	// Used with Datadog APM
	HeaderXTraceID = "x-trace-id"
)

// Authentication Headers
const (
	// HeaderAuthorization is the standard HTTP header used to carry authentication
	// credentials such as Bearer tokens, Basic auth, or API keys
	// Format examples: "Bearer <token>", "Basic <base64-encoded-credentials>"
	HeaderAuthorization = "authorization"
)

// Client Identification Headers
const (
	// HeaderClientTaggingHeader is used to identify and tag requests from specific
	// clients or applications. This allows for client-specific analytics,
	// rate limiting, feature flags, and monitoring in multi-tenant systems
	HeaderClientTaggingHeader = "x-client-id"
)

// HeaderConfig defines which headers to extract and forward
type HeaderConfig struct {
	// HeadersToForward specifies which HTTP headers to forward as metadata
	HeadersToForward []string
}

func GetHeadersToForward() []string {
	return []string{
		HeaderXRequestID,
		HeaderXCorrelationID,
		HeaderXTraceID,
		HeaderAuthorization,
		HeaderClientTaggingHeader,
	}
}
