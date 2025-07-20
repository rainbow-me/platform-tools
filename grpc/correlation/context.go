package correlation

import (
	"context"
	"strings"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
)

// Standard correlation keys
const (
	TenancyKey = "tenancy"
	IDKey      = "correlation_id"
)

// ContextCorrelationHeader HTTP/gRPC header name for correlation context
const ContextCorrelationHeader = internalmetadata.HeaderXCorrelationID

// correlationContextKey is a private type for context keys to avoid collisions
type correlationContextKey struct{}

// Key CorrelationKey is the context key for storing correlation data
var Key = correlationContextKey{} //nolint:gochecknoglobals

// Data CorrelationData represents the correlation context data
type Data map[string]string

// Set adds correlation values to the context and returns a new context.
// This function is thread-safe as it creates a new map.
func Set(ctx context.Context, values map[string]string) context.Context {
	if len(values) == 0 {
		return ctx
	}

	// Create a new map to avoid mutation issues
	correlationMap := make(Data, len(values))
	for k, v := range values {
		if k != "" && v != "" { // Only store non-empty key-value pairs
			correlationMap[k] = v
		}
	}

	// Set baggage items for distributed tracing
	if span, ok := tracer.SpanFromContext(ctx); ok {
		for k, v := range correlationMap {
			span.SetBaggageItem(k, v)
		}
	}

	return context.WithValue(ctx, Key, correlationMap)
}

// SetKey sets a single correlation key-value pair and returns a new context.
func SetKey(ctx context.Context, key, value string) context.Context {
	if key == "" {
		return ctx
	}

	existing := Get(ctx)
	newMap := make(Data, len(existing)+1)

	// Copy existing values
	for k, v := range existing {
		newMap[k] = v
	}

	// Set/update the new value
	if value != "" {
		newMap[key] = value
	} else {
		delete(newMap, key) // Remove empty values
	}

	// Set baggage item for distributed tracing
	if span, ok := tracer.SpanFromContext(ctx); ok {
		span.SetBaggageItem(key, value)
	}

	return context.WithValue(ctx, Key, newMap)
}

// Get returns the correlation data from the context.
// Returns an empty map if no correlation data exists.
func Get(ctx context.Context) Data {
	if ctx == nil {
		return make(Data)
	}

	if v, ok := ctx.Value(Key).(Data); ok && v != nil {
		return v
	}

	return make(Data)
}

// GetValue returns a specific correlation value by key.
// Returns empty string if the key doesn't exist.
func GetValue(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	return Get(ctx)[key]
}

// Has checks if a correlation key exists in the context.
func Has(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}
	_, exists := Get(ctx)[key]
	return exists
}

// Delete removes a correlation key from the context and returns a new context.
func Delete(ctx context.Context, key string) context.Context {
	if key == "" {
		return ctx
	}

	existing := Get(ctx)
	if _, exists := existing[key]; !exists {
		return ctx // Key doesn't exist, no change needed
	}

	newMap := make(Data, len(existing))
	for k, v := range existing {
		if k != key {
			newMap[k] = v
		}
	}

	return context.WithValue(ctx, Key, newMap)
}

// Merge combines correlation data from multiple contexts.
// Later contexts override values from earlier ones.
func Merge(ctx context.Context, otherContexts ...context.Context) context.Context {
	if len(otherContexts) == 0 {
		return ctx
	}

	merged := make(Data)

	// Start with base context
	for k, v := range Get(ctx) {
		merged[k] = v
	}

	// Merge other contexts
	for _, otherCtx := range otherContexts {
		for k, v := range Get(otherCtx) {
			if v != "" {
				merged[k] = v
			}
		}
	}

	return Set(ctx, merged)
}

// ToZapFields converts the correlation context to zap fields for logging.
func ToZapFields(ctx context.Context) []zap.Field {
	data := Get(ctx)
	if len(data) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(data))
	for key, value := range data {
		if value != "" {
			fields = append(fields, zap.String(key, value))
		}
	}

	return fields
}

// ToMap returns a copy of the correlation data as a regular map.
func ToMap(ctx context.Context) map[string]string {
	data := Get(ctx)
	result := make(map[string]string, len(data))
	for k, v := range data {
		result[k] = v
	}
	return result
}

// Tenancy returns the tenancy value from the correlation context.
// Returns empty string if not found.
func Tenancy(ctx context.Context) string {
	return GetValue(ctx, TenancyKey)
}

// SetTenancy sets the tenancy value in the correlation context.
func SetTenancy(ctx context.Context, tenancy string) context.Context {
	return SetKey(ctx, TenancyKey, tenancy)
}

// ID returns the correlation ID from the correlation context.
// Returns empty string if not found.
func ID(ctx context.Context) string {
	return GetValue(ctx, IDKey)
}

// SetID sets the correlation ID in the correlation context.
func SetID(ctx context.Context, correlationID string) context.Context {
	return SetKey(ctx, IDKey, correlationID)
}

// Keys returns all correlation keys present in the context.
func Keys(ctx context.Context) []string {
	data := Get(ctx)
	if len(data) == 0 {
		return nil
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

// IsEmpty returns true if the correlation context has no data.
func IsEmpty(ctx context.Context) bool {
	return len(Get(ctx)) == 0
}

// String returns a string representation of the correlation data.
func String(ctx context.Context) string {
	data := Get(ctx)
	if len(data) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(data))
	for k, v := range data {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, ",")
}

// Generate creates the correlation header string from the context.
func Generate(ctx context.Context) string {
	return String(ctx)
}

// ParseCorrelationHeader parses the correlation header string into a Data map.
func ParseCorrelationHeader(headerVal string) Data {
	pairs := strings.Split(headerVal, ",")
	m := make(Data)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if k != "" && v != "" {
				m[k] = v
			}
		}
	}
	return m
}

// GetFirst returns the first value for a metadata key, or an empty string if not present.
func GetFirst(md metadata.MD, key string) string {
	val := md.Get(key)
	if len(val) > 0 {
		return val[0]
	}
	return ""
}
