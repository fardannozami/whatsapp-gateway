package http

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type requestIDKey struct{}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx := context.WithValue(c.Request.Context(), requestIDKey{}, requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID := RequestIDFromContext(c.Request.Context())
		if requestID == "" {
			requestID = c.GetString("request_id")
		}

		log.Printf("request_id=%s method=%s path=%s status=%d latency=%s", requestID, c.Request.Method, c.Request.URL.Path, status, latency)
	}
}
