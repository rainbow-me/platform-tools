package metadata

// RequestInfo contains extracted information from gRPC metadata
type RequestInfo struct {
	RequestTime string `json:"requestTime"`

	// Request Identification
	RequestID     string `json:"requestId"`
	CorrelationID string `json:"correlationId"`
	TraceID       string `json:"traceId"`

	// Authentication
	HasAuth   bool   `json:"hasAuth"`
	AuthType  string `json:"authType,omitempty"`  // e.g., Bearer, Basic, Digest, ApiKey
	AuthToken string `json:"authToken,omitempty"` // Masked if sensitive

	// Raw headers for debugging
	AllHeaders map[string]string `json:"allHeaders,omitempty"`
}
