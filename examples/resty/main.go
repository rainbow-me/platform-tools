package main

import (
	"context"
	"log"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/go-resty/resty/v2"

	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/common/metadata"
	metadata2 "github.com/rainbow-me/platform-tools/grpc/metadata"
	restyinterceptors "github.com/rainbow-me/platform-tools/http/interceptors/resty"
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

	client := resty.New()
	restyinterceptors.InjectInterceptors(client)

	span := tracer.StartSpan("ping.request")
	ctx := tracer.ContextWithSpan(context.Background(), span)
	ctx = metadata.ContextWithRequestInfo(ctx, metadata2.RequestInfo{
		RequestTime:   "",
		RequestID:     "",
		CorrelationID: "",
		TraceID:       "",
		HasAuth:       false,
		AuthType:      "",
		AuthToken:     "",
		AllHeaders:    nil,
	})

	l, err := logger.Instance()
	if err != nil {
		return err
	}
	l.Info("Sending ping request", logger.String("trace_id", span.Context().TraceID()))

	_, err = client.R().SetContext(ctx).Get("http://localhost:8080/ping")
	return err
}
