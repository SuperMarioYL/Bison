package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/pkg/logger"
)

// bodyLogWriter wraps gin.ResponseWriter to capture response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger returns a gin middleware that logs requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method

		// Read request body for logging (only for non-GET requests)
		var requestBody string
		if method != "GET" && method != "HEAD" {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			// Restore body for handler
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Wrap response writer to capture body
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()

		// Build log fields
		fields := []interface{}{
			"status", status,
			"method", method,
			"path", path,
			"latency", latency.String(),
			"ip", clientIP,
		}

		if query != "" {
			fields = append(fields, "query", query)
		}

		// Log based on status code
		if status >= 500 {
			// Server error - log with request and response body
			fields = append(fields, "request_body", truncateString(requestBody, 1000))
			fields = append(fields, "response_body", truncateString(blw.body.String(), 500))
			if len(c.Errors) > 0 {
				fields = append(fields, "errors", c.Errors.String())
			}
			logger.Error("request failed", fields...)
		} else if status >= 400 {
			// Client error - log with request body
			fields = append(fields, "request_body", truncateString(requestBody, 500))
			fields = append(fields, "response_body", truncateString(blw.body.String(), 200))
			logger.Warn("client error", fields...)
		} else {
			// Success
			logger.Info("request completed", fields...)
		}
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Recovery returns a gin middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal server error",
					"code":  "INTERNAL_ERROR",
				})
			}
		}()
		c.Next()
	}
}

