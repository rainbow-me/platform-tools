package interceptors

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/mennanov/fmutils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/rainbow-me/platform-tools/grpc/correlation"
	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
)

// Structured logging field keys
const (
	// Request timing and identification
	durationDDKey = "duration"
	traceIDKey    = "trace_id"
	spanIDKey     = "span_id"

	// Request context information
	isNewTraceKey = "is_new_trace"
	clientIDKey   = "client_id"
	serviceKey    = "service"
	methodKey     = "method"
	grpcStatuKey  = "status"

	// Request/response payloads
	requestKey  = "request"
	responseKey = "response"

	// Client identification header
	clientTaggingHeader = internalmetadata.HeaderClientTaggingHeader
)

// Pre-compile regex for performance - avoid recompiling on each request
var methodRegex = regexp.MustCompile(`\/(.+)\/(.+)$`)

// logWithContext logs gRPC calls with comprehensive context information,
// optionally including request and response payloads based on configuration.
// This function handles the complete lifecycle of a gRPC request logging.
func logWithContext(
	ctx context.Context,
	at string,
	fullMethod string,
	config *LoggingInterceptorConfig,
	log *zap.Logger,
	req interface{},
	handler func(ctx context.Context) (interface{}, error),
) (interface{}, error) {
	// Skip logging if method is in the skip list
	if _, shouldSkip := config.skipLoggingByMethod[fullMethod]; shouldSkip {
		return handler(ctx)
	}

	// Disable stack traces for all levels since interceptor stacks are not useful
	log = log.WithOptions(zap.AddStacktrace(zap.ErrorLevel + 1))

	// Extract service and method names from the full gRPC method path
	grpcService, grpcMethod := GetServiceAndMethod(fullMethod)

	// Build base log fields that will be included in all log entries
	baseLogFields := buildBaseLogFields(ctx, grpcService, grpcMethod)

	// Add logger with base fields to context for downstream use
	ctx = ctxzap.ToContext(ctx, log.With(baseLogFields...))

	// Execute the gRPC handler and measure execution time
	startTime := time.Now()
	resp, err := handler(ctx)

	// Skip logging if disabled and no error occurred
	if !config.LogEnabled && err == nil {
		return resp, nil
	}

	executionDuration := time.Since(startTime)

	// Build log fields for this specific request
	logFields := buildRequestLogFields(config, req, resp, executionDuration)

	// Determine log level based on error status
	logLevel := determineLogLevel(config, err)

	// Add gRPC status and error information
	logFields = append(logFields, buildStatusLogFields(err)...)

	// Add client and trace information from metadata
	logFields = append(logFields, buildMetadataLogFields(ctx)...)

	// Write the log entry
	ctxzap.Extract(ctx).Check(logLevel, at).Write(logFields...)

	return resp, err
}

// buildBaseLogFields creates the base log fields that are common to all requests
func buildBaseLogFields(ctx context.Context, grpcService, grpcMethod string) []zapcore.Field {
	var fields []zapcore.Field

	// Add trace information if available
	if span, ok := tracer.SpanFromContext(ctx); ok {
		fields = append(fields,
			zap.String(traceIDKey, span.Context().TraceID()),
			zap.String(spanIDKey, strconv.FormatUint(span.Context().SpanID(), 10)),
		)
	}

	// Add correlation fields and service/method information
	fields = append(fields, correlation.ToZapFields(ctx)...)
	fields = append(fields,
		zap.String(methodKey, grpcMethod),
		zap.String(serviceKey, grpcService),
	)

	return fields
}

// buildRequestLogFields creates log fields for request/response data and timing
func buildRequestLogFields(
	config *LoggingInterceptorConfig,
	req, resp interface{},
	duration time.Duration,
) []zapcore.Field {
	var fields []zapcore.Field

	// Add request payload if logging is enabled
	if config.LogParams || config.LogRequests {
		fields = append(fields, GrpcMessageField(requestKey, req, config.LogParamsBlocklist))
	}

	// Always add execution duration
	fields = append(fields, zap.Duration(durationDDKey, duration))

	// Add response payload if logging is enabled and response is not nil
	if (config.LogParams || config.LogResponses) && resp != nil && !reflect.ValueOf(resp).IsZero() {
		fields = append(fields, GrpcMessageField(responseKey, resp, config.LogParamsBlocklist))
	}

	return fields
}

// determineLogLevel determines the appropriate log level based on error status
func determineLogLevel(config *LoggingInterceptorConfig, err error) zapcore.Level {
	if err == nil {
		return config.LogLevel
	}

	// Check if there's a specific log level configured for this gRPC status code
	statusValue := status.Convert(err)
	if codeLevel, exists := config.GrpcCodeLogLevel[statusValue.Code()]; exists {
		return codeLevel
	}

	// Default to error log level for errors
	return config.ErrorLogLevel
}

// buildStatusLogFields creates log fields for gRPC status and error information
func buildStatusLogFields(err error) []zapcore.Field {
	var fields []zapcore.Field

	statusValue := status.Convert(err)
	fields = append(fields, zap.String(grpcStatuKey, statusValue.Code().String()))

	// Add error details if present
	if err != nil {
		fields = append(fields, zap.Error(err))
	}

	return fields
}

// buildMetadataLogFields extracts client and trace information from gRPC metadata
func buildMetadataLogFields(ctx context.Context) []zapcore.Field {
	var fields []zapcore.Field

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fields
	}

	// Extract client ID from metadata
	clientIDs := md.Get(clientTaggingHeader)
	if len(clientIDs) > 0 {
		fields = append(fields, zap.String(clientIDKey, clientIDs[0]))
	} else {
		fields = append(fields, zap.String(clientIDKey, "unknown"))
	}

	// Check if this is a new trace (no incoming trace ID)
	traceIDs := md.Get(tracer.DefaultTraceIDHeader)
	fields = append(fields, zap.Bool(isNewTraceKey, len(traceIDs) == 0))

	return fields
}

// GrpcMessageField creates a zap field for gRPC messages with optional field masking.
// It clones the message to avoid modifying the original and applies any configured masks.
func GrpcMessageField(key string, message interface{}, masks []fieldmaskpb.FieldMask) zapcore.Field {
	msg, ok := message.(proto.Message)
	if !ok {
		return PbField(key, message)
	}

	// Clone the message to avoid modifying the original
	clonedMsg := proto.Clone(msg)

	// Apply field masks to remove sensitive information
	for i := range masks {
		clonedMsg = pruneFields(clonedMsg, &masks[i])
	}

	return PbField(key, clonedMsg)
}

// pruneFields removes specified fields from a protobuf message based on field mask.
// This is used to remove sensitive information from logs.
func pruneFields(message proto.Message, mask *fieldmaskpb.FieldMask) proto.Message {
	if mask != nil {
		fmutils.Prune(message, mask.GetPaths())
	}
	return message
}

// GetServiceAndMethod extracts the service and method names from a full gRPC method path.
// Input format: "/rainbow.rates.Rates/Spot"
// Output: service="rainbow.rates.Rates", method="Spot"
func GetServiceAndMethod(fullMethod string) (string, string) {
	// Use pre-compiled regex to extract service and method
	methodParts := methodRegex.FindStringSubmatch(fullMethod)
	var service, method string
	if len(methodParts) >= 3 {
		service = methodParts[1]
		method = methodParts[2]
	} else {
		// Fallback: use the full method if parsing fails
		method = fullMethod
		service = "unknown"
	}

	return service, method
}

// PbField wraps a protobuf message in a zap Field for structured logging.
// Use this to embed protobuf messages in your structured zap logs.
func PbField(key string, pb interface{}) zapcore.Field {
	if pbMsg, ok := pb.(proto.Message); ok {
		return zap.Object(key, &pbZapField{pbMsg})
	}

	// Fallback for non-protobuf messages
	return zap.Any(key, pb)
}

// pbZapField is a wrapper that implements zapcore.ObjectMarshaler
// for protobuf messages in structured logging
type pbZapField struct {
	pb proto.Message
}

// MarshalLogObject implements zapcore.ObjectMarshaler for structured logging
func (p *pbZapField) MarshalLogObject(e zapcore.ObjectEncoder) error {
	return e.AddReflected("payload", p)
}

// MarshalJSON implements json.Marshaler for JSON log output
func (p *pbZapField) MarshalJSON() ([]byte, error) {
	b, err := protojson.Marshal(p.pb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf message to JSON: %w", err)
	}
	return b, nil
}
