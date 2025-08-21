package gin

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/logger"
)

type loggingCfg struct {
	debug bool
	trace bool
}

type responseWriterCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriterCapture) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func RequestLogging(cfg loggingCfg) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var reqBody, respBody []byte

		// Capture the request body if trace logging is enabled
		if cfg.trace && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				reqBody = bodyBytes
				// Restore the request body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
		c.Request = c.Request.WithContext(ctx)

		// Record start time for duration calculation
		start := time.Now()

		// Set up response body capture if trace logging is enabled
		var responseCapture *responseWriterCapture
		if cfg.trace {
			responseCapture = &responseWriterCapture{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
			}
			c.Writer = responseCapture
		}

		// Proceed to the next middleware or handler
		c.Next()

		// Extract response body if it was captured
		if cfg.trace && responseCapture != nil {
			respBody = responseCapture.body.Bytes()
		}

		if cfg.debug {
			// After handling, build log fields
			duration := time.Since(start)
			var fields []logger.Field
			fields = append(fields,
				logger.String("method", c.Request.Method),
				logger.String("path", c.Request.URL.Path),
				logger.Int("status", c.Writer.Status()),
				logger.Duration("duration", duration),
				logger.String("component", "gin"),
			)

			if cfg.trace {
				fields = append(fields,
					logger.ByteString("request_body", reqBody),
					logger.ByteString("response_body", respBody),
				)
			}

			// Determine log level based on status code
			logLevel := logger.DebugLevel
			if c.Writer.Status() >= 500 {
				logLevel = logger.ErrorLevel
			} else if c.Writer.Status() >= 400 {
				logLevel = logger.WarnLevel
			}
			logger.FromContext(ctx).Log(logLevel, "HTTP request handled", fields...)
		}
	}
}
