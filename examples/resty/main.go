package main

import (
	"context"
	"log"
	gohttp "net/http"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"

	"github.com/rainbow-me/platform-tools/common/correlation"
	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/common/metadata"
	"github.com/rainbow-me/platform-tools/http"
	"github.com/rainbow-me/platform-tools/observability"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := tracer.Start(tracer.WithService("resty-playground")); err != nil {
		return err
	}
	defer tracer.Stop()

	l, err := logger.Instance()
	if err != nil {
		return err
	}

	client := http.NewRestyWithClient(gohttp.DefaultClient, l)

	span, ctx := observability.StartSpan(context.Background(), "ping.request")
	ctx = correlation.SetKey(ctx, "my-key", "my-value")
	ctx = metadata.ContextWithRequestInfo(ctx, metadata.RequestInfo{RequestID: "my-request-id"})

	l = logger.FromContext(ctx) // ensure it contains request info
	l.Info("Sending ping request", logger.String("trace_id", span.Context().TraceID()))

	_, err = client.R().SetContext(ctx).SetBody(struct {
		Message string `json:"message"`
	}{Message: "ping"}).
		Post("http://localhost:8080/ping")
	return err
}
