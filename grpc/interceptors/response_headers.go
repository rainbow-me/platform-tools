package interceptors

import (
	"context"
	"strconv"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
)

// ResponseHeadersInterceptor adds trace and request ID headers to gRPC responses
func ResponseHeadersInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract headers before calling the handler
		traceID, spanID := extractDataDogIDs(ctx)
		rainbowRequestID := extractRainbowRequestID(ctx)

		// Call the actual handler
		resp, err := handler(ctx, req)

		// Add headers to response metadata
		addResponseHeaders(ctx, traceID, spanID, rainbowRequestID)

		return resp, err
	}
}

// extractDataDogIDs extracts trace ID and span ID from DataDog context
func extractDataDogIDs(ctx context.Context) (string, string) {
	span, ok := tracer.SpanFromContext(ctx)
	if !ok {
		return "", ""
	}

	spanContext := span.Context()
	traceID := spanContext.TraceID()
	spanID := strconv.FormatUint(spanContext.SpanID(), 10)

	return traceID, spanID
}

// extractRainbowRequestID extracts rainbow request ID from incoming metadata
func extractRainbowRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	// Try different possible header names for rainbow request ID
	possibleKeys := []string{
		internalmetadata.HeaderXRequestID,
	}

	for _, key := range possibleKeys {
		if values := md.Get(key); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}

	return ""
}

// addResponseHeaders adds headers to the outgoing response metadata
func addResponseHeaders(ctx context.Context, traceID, spanID, rainbowRequestID string) {
	// Prepare mdHeaders to send
	mdHeaders := metadata.Pairs()

	// Add DataDog trace mdHeaders if available
	if traceID != "" {
		mdHeaders = metadata.Join(mdHeaders, metadata.Pairs(internalmetadata.HeaderXTraceID, traceID))
	}

	if spanID != "" {
		mdHeaders = metadata.Join(mdHeaders, metadata.Pairs(internalmetadata.HeaderXSpanID, spanID))
	}

	// Add rainbow request ID if available
	if rainbowRequestID != "" {
		mdHeaders = metadata.Join(mdHeaders, metadata.Pairs(internalmetadata.HeaderXRequestID, rainbowRequestID))
	}

	// Send mdHeaders to client
	if len(mdHeaders) > 0 {
		err := grpc.SendHeader(ctx, mdHeaders)
		if err != nil {
			return
		}
	}
}
