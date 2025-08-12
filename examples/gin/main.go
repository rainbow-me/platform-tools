package main

import (
	"log"
	"net/http"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/logger"
	gininterceptors "github.com/rainbow-me/platform-tools/http/interceptors/gin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := tracer.Start(tracer.WithService("gin-playground")); err != nil {
		return err
	}
	defer tracer.Stop()

	r := gin.New()

	r.Use(gininterceptors.DefaultInterceptors()...)

	r.GET("/ping", func(c *gin.Context) {
		span, ok := tracer.SpanFromContext(c.Request.Context())
		if ok {
			span.SetTag("custom.tag", "example-value")
		}
		logger.FromContext(c.Request.Context()).Info("Received ping request")

		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/error", func(c *gin.Context) {
		// this error will be logged by our middleware as 'some-error' and returned as an 'internal server error' in the response
		_ = c.Error(errors.New("some-error")) // this should also include a stack trace
		return
	})

	r.GET("/panic", func(c *gin.Context) {
		// panic will be handled as internal server error
		panic("boom")
	})

	return r.Run(":8080")
}
