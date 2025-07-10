package correlation

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Parse converts a string to a correlation context and adds it to the context.
//
//	Unsafe to use concurrently.
func Parse(ctx context.Context, header string) context.Context {
	if header == "" {
		return ctx
	}

	kvs := strings.Split(header, ",")
	correlationContext := make(map[string]string, len(kvs))

	for i := range kvs {
		kv := strings.Split(kvs[i], "=")

		if correlationContext[kv[0]] != "" {
			continue
		}

		if len(kv) == 1 {
			correlationContext[kv[0]] = "true"
		} else {
			unescaped, err := url.QueryUnescape(kv[1])
			if err != nil {
				unescaped = kv[1]
			}

			correlationContext[kv[0]] = unescaped
		}
	}

	return Set(ctx, correlationContext)
}

// Generate converts a correlation context to a string.
//
//	Unsafe to use concurrently.
func Generate(ctx context.Context) string {
	var kvs []string

	for k, v := range Get(ctx) {
		escaped := url.QueryEscape(v)
		kvs = append(kvs, fmt.Sprintf("%s=%s", k, escaped))
	}

	return strings.Join(kvs, ",")
}
