package correlation

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rainbow-me/platform-tools/common/headers"
	"github.com/rainbow-me/platform-tools/common/logger"
)

// Standard correlation keys
const (
	TenancyKey       = "tenancy"
	IDKey            = "correlation_id"
	IdempotencyKeyID = "idempotency_key"
)

// ContextCorrelationHeader HTTP/gRPC header name for correlation context
const ContextCorrelationHeader = headers.HeaderXCorrelationData

// correlationContextKey is a private type for context keys to avoid collisions
type correlationContextKey struct{}

// Key CorrelationKey is the context key for storing correlation data
var Key = correlationContextKey{}

// Data CorrelationData represents the correlation context data
type Data map[string]string

func ContextWithCorrelation(ctx context.Context, val string) context.Context {
	if val != "" {
		correlationData, err := ParseCorrelationHeader(val)
		if err != nil {
			logger.FromContext(ctx).Warn("failed to parse correlation header", zap.Error(err))
		} else {
			ctx = Set(ctx, correlationData)
		}
	}
	// Generate correlation_id if missing
	if !Has(ctx, IDKey) {
		ctx = SetID(ctx, uuid.NewString())
	}
	return ctx
}

// Set adds correlation values to a context, returning a new context.
// - Derives new context; doesn't modify input context or values map.
// - Merges input values into a copy of existing context correlation data.
// - Safe for concurrent calls on shared context; each call is independent.
// - Requires external sync if input map is concurrently modified.
// - Stored map is mutable; treat as read-only to avoid issues.
func Set(ctx context.Context, values map[string]string) context.Context {
	if len(values) == 0 {
		return ctx
	}

	// Clone existing correlation data from context
	// no need to check for nil, maps.Clone handles it, get returns empty map if nil
	correlationMap := maps.Clone(Get(ctx))

	// Update with non-empty key-value pairs from input
	for k, v := range values {
		if k != "" && v != "" {
			correlationMap[k] = v
		}
	}

	// Set baggage items for distributed tracing
	if span, ok := tracer.SpanFromContext(ctx); ok {
		for k, v := range correlationMap {
			span.SetBaggageItem(k, v)
		}
	}

	ctx = context.WithValue(ctx, Key, correlationMap)
	return logger.ContextWithFields(ctx, toLogFields(correlationMap)...)
}

// SetKey sets a single correlation key-value pair and returns a new context.
// - Derives new context; doesn't modify input context.
// - Merges the key-value into a copy of existing context correlation data.
// - Safe for concurrent calls on shared context; each call is independent.
// - Stored map is mutable; treat as read-only to avoid issues.
func SetKey(ctx context.Context, key, value string) context.Context {
	if key == "" {
		return ctx
	}

	// Clone existing correlation data from context
	// no need to check for nil, maps.Clone handles it, get returns empty map if nil
	newMap := maps.Clone(Get(ctx))

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

	ctx = context.WithValue(ctx, Key, newMap)
	return logger.ContextWithFields(ctx, toLogFields(newMap)...)
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
// - Derives new context; doesn't modify input context.
// - Safe for concurrent calls on shared context; each call is independent.
// - Stored map is mutable; treat as read-only to avoid issues.
func Delete(ctx context.Context, key string) context.Context {
	if key == "" {
		return ctx
	}

	existing := Get(ctx)
	if _, exists := existing[key]; !exists {
		return ctx // Key doesn't exist, no change needed
	}

	newMap := maps.Clone(existing)
	delete(newMap, key)

	return context.WithValue(ctx, Key, newMap)
}

// Merge combines correlation data from multiple contexts.
// - Later contexts override values from earlier ones.
// - Derives new context; doesn't modify input context.
// - Safe for concurrent calls on shared context; each call is independent.
// - Stored map is mutable; treat as read-only to avoid issues.
func Merge(ctx context.Context, otherContexts ...context.Context) context.Context {
	if len(otherContexts) == 0 {
		return ctx
	}

	// Clone base context correlation data
	merged := maps.Clone(Get(ctx))

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

// ToLogFields converts the correlation context to zap fields for logging.
func ToLogFields(ctx context.Context) []logger.Field {
	return toLogFields(Get(ctx))
}

// toLogFields converts the correlation context to zap fields for logging.
func toLogFields(data Data) []logger.Field {
	if len(data) == 0 {
		return nil
	}

	fields := make([]logger.Field, 0, len(data))
	for key, value := range data {
		if value != "" {
			fields = append(fields, logger.String(key, value))
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

// IdempotencyKey returns the idempotency key from the correlation context.
// Returns empty string if not found.
func IdempotencyKey(ctx context.Context) string {
	return GetValue(ctx, IdempotencyKeyID)
}

// SetIdempotencyKey sets the idempotency key in the correlation context.
func SetIdempotencyKey(ctx context.Context, key string) context.Context {
	return SetKey(ctx, IdempotencyKeyID, key)
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
		return "{}"
	}
	j, _ := json.Marshal(data)
	return string(j)
}

// Generate creates the correlation header string from the context.
func Generate(ctx context.Context) string {
	return String(ctx)
}

// ParseCorrelationHeader parses the correlation header string into a Data map.
func ParseCorrelationHeader(headerVal string) (Data, error) {
	var data Data
	err := json.Unmarshal([]byte(headerVal), &data)
	return data, err
}
