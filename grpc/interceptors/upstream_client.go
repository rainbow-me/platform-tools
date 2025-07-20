package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
)

const UpstreamServiceHeaderKey = internalmetadata.HeaderClientTaggingHeader

// UnaryUpstreamInfoClientInterceptor creates a new interceptor with the given server name
func UnaryUpstreamInfoClientInterceptor(serverName string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Extract service name from the method (format: /package.Service/Method)
		serviceName := serverName
		if serviceName == "" {
			// Fallback to extracting from method if serverName not provided
			parts := strings.Split(method, "/")
			if len(parts) >= 2 {
				serviceName = strings.Split(parts[1], ".")[len(strings.Split(parts[1], "."))-1]
			}
		}

		ctx = metadata.AppendToOutgoingContext(ctx, UpstreamServiceHeaderKey, serviceName)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
