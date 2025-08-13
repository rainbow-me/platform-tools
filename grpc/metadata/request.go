package metadata

import (
	"context"
	"strings"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	"github.com/rainbow-me/platform-tools/common/headers"
	commonmeta "github.com/rainbow-me/platform-tools/common/metadata"
)

// Parser extracts commonmeta.RequestInfo from gRPC metadata
type Parser struct {
	opt RequestParserOpt
}

type RequestParserOpt struct {
	IncludeAllHeaders bool
	MaskSensitive     bool
}

// NewRequestParser creates a new metadata parser
func NewRequestParser(opt RequestParserOpt) *Parser {
	return &Parser{
		opt: opt,
	}
}

// ParseMetadata extracts commonmeta.RequestInfo from metadata and ensures
// that essential request identifiers are always present by generating
// them ONLY when not provided by the client. If a request ID is generated,
// it is added to the metadata for downstream propagation.
//
// This function:
// - Extracts request/correlation/trace IDs from metadata
// - Generates a request ID ONLY if none is present in headers
// - Adds generated request ID to metadata for propagation (only if generated)
// - Extracts authentication information
// - Optionally includes all headers based on configuration
//
// Parameters:
//   - ctx: context containing metadata
//
// Returns:
//   - context.Context: Updated context with request ID in metadata (only if request ID was generated)
//   - *commonmeta.RequestInfo: Populated request information with guaranteed request ID
func (p *Parser) ParseMetadata(ctx context.Context) (context.Context, *commonmeta.RequestInfo) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// No metadata present, create new metadata with generated request ID
		requestID := p.generateRequestID()
		info := &commonmeta.RequestInfo{
			RequestTime: time.Now().Format(time.RFC3339),
			RequestID:   requestID,
		}

		// Create new metadata with the generated request ID
		newMD := metadata.New(map[string]string{
			headers.HeaderXRequestID: requestID,
		})
		updatedCtx := metadata.NewIncomingContext(ctx, newMD)

		return updatedCtx, info
	}

	info := &commonmeta.RequestInfo{
		RequestTime: time.Now().Format(time.RFC3339),
	}

	// Extract all the information (request ID generation and metadata update happens ONLY if needed)
	updatedCtx, requestIDGenerated := p.extractRequestIDs(ctx, md, info)
	p.extractAuthentication(md, info)

	// Include all headers if requested
	if p.opt.IncludeAllHeaders {
		p.extractAllHeaders(md, info)
	}

	// Return original context if no request ID was generated, updated context if it was
	if requestIDGenerated {
		return updatedCtx, info
	}

	return ctx, info
}

// extractRequestIDs extracts various request/trace IDs from metadata.
// ONLY generates and adds a request ID to metadata if none is found in the headers.
// Returns the updated context and a boolean indicating if a request ID was generated.
func (p *Parser) extractRequestIDs(
	ctx context.Context,
	md metadata.MD,
	info *commonmeta.RequestInfo,
) (context.Context, bool) {
	// Check common request ID headers
	requestIDHeaders := []string{
		headers.HeaderXRequestID,
	}

	// Try to extract request ID from headers
	for _, header := range requestIDHeaders {
		if value := p.getFirstValue(md, header); value != "" {
			info.RequestID = value
			break
		}
	}

	// ONLY generate request ID and add to metadata if none was found in headers
	var updatedCtx context.Context
	var requestIDGenerated bool

	if info.RequestID == "" {
		// Generate request ID only when not present
		info.RequestID = p.generateRequestID()
		requestIDGenerated = true

		// Add the generated request ID to the metadata
		updatedCtx = p.addRequestIDToMetadata(ctx, md, info.RequestID)
	} else {
		// Request ID was found in headers, no need to update context
		updatedCtx = ctx
		requestIDGenerated = false
	}

	// Check correlation ID headers
	correlationHeaders := []string{
		headers.HeaderXCorrelationID,
	}

	for _, header := range correlationHeaders {
		if value := p.getFirstValue(md, header); value != "" {
			info.CorrelationID = value
			break
		}
	}

	// Extract trace ID from Datadog tracer if available
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		info.TraceID = span.Context().TraceID()
	}

	return updatedCtx, requestIDGenerated
}

// addRequestIDToMetadata adds the generated request ID to the gRPC metadata
// so it can be propagated to downstream services and accessed by other interceptors.
func (p *Parser) addRequestIDToMetadata(
	ctx context.Context,
	existingMD metadata.MD,
	requestID string,
) context.Context {
	// Clone the existing metadata to avoid modifying the original
	newMD := existingMD.Copy()

	// Add the request ID to the metadata
	newMD.Set(headers.HeaderXRequestID, requestID)

	// Create a new context with the updated metadata
	return metadata.NewIncomingContext(ctx, newMD)
}

// generateRequestID creates a new unique request identifier.
// Uses UUID v4 to ensure uniqueness across distributed systems.
// Returns a string representation of the generated UUID.
func (p *Parser) generateRequestID() string {
	// Generate a new UUID for the request ID
	requestUUID := uuid.New()
	return requestUUID.String()
}

// extractAuthentication extracts authentication information
func (p *Parser) extractAuthentication(md metadata.MD, info *commonmeta.RequestInfo) {
	// Authorization header
	if authHeader := p.getFirstValue(md, headers.HeaderAuthorization); authHeader != "" {
		info.HasAuth = true
		info.AuthType = p.extractAuthType(authHeader)

		if !p.opt.MaskSensitive {
			info.AuthToken = authHeader
		} else {
			info.AuthToken = p.maskToken(authHeader)
		}
	}
}

// extractAllHeaders extracts all headers for debugging
func (p *Parser) extractAllHeaders(md metadata.MD, info *commonmeta.RequestInfo) {
	info.AllHeaders = make(map[string]string)

	for key, values := range md {
		if len(values) > 0 {
			// Mask sensitive headers
			if p.opt.MaskSensitive && p.isSensitiveHeader(key) {
				info.AllHeaders[key] = p.maskToken(values[0])
			} else {
				info.AllHeaders[key] = values[0]
			}
		}
	}
}

// Helper functions
func (p *Parser) getFirstValue(md metadata.MD, keys ...string) string {
	for _, key := range keys {
		if values := md.Get(key); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	return ""
}

func (p *Parser) extractAuthType(authHeader string) string {
	authHeader = strings.ToLower(authHeader)

	switch {
	case strings.HasPrefix(authHeader, "bearer "):
		return "Bearer"
	case strings.HasPrefix(authHeader, "basic "):
		return "Basic"
	case strings.HasPrefix(authHeader, "digest "):
		return "Digest"
	case strings.HasPrefix(authHeader, "apikey "):
		return "ApiKey"
	default:
		return "Unknown"
	}
}

func (p *Parser) maskToken(token string) string {
	if len(token) <= 10 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}

func (p *Parser) isSensitiveHeader(key string) bool {
	sensitiveHeaders := []string{
		headers.HeaderAuthorization,
	}

	key = strings.ToLower(key)
	for _, sensitive := range sensitiveHeaders {
		if key == sensitive {
			return true
		}
	}
	return false
}

// GetFirst returns the first value for a metadata key, or an empty string if not present.
func GetFirst(md metadata.MD, key string) string {
	val := md.Get(key)
	if len(val) > 0 {
		return val[0]
	}
	return ""
}
