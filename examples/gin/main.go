package main

import (
	"log"
	"net/http"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
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

	// Your handlers will automatically have tracing context
	r.GET("/ping", func(c *gin.Context) {
		// The span is automatically created by the middleware
		// and available in the context
		span, ok := tracer.SpanFromContext(c.Request.Context())
		if ok {
			span.SetTag("custom.tag", "example-value")
		}
		logger.FromContext(c.Request.Context()).Info("Received ping request")

		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	return r.Run(":8080")
}
