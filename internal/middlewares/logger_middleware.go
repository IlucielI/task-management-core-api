package middlewares

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"task-management/internal/pkg/ctxmeta"
)

// LogEntry defines the structured JSON log output format.
type LogEntry struct {
	Level      string `json:"level"`
	Timestamp  string `json:"timestamp"`
	RequestID  string `json:"request_id"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Latency    string `json:"latency"`
	LatencyMs  int64  `json:"latency_ms"`
	ClientIP   string `json:"client_ip,omitempty"`
}

type loggerOptions struct {
	writer io.Writer
}

// LoggerOption configures the StructuredLogger.
type LoggerOption func(*loggerOptions)

// WithWriter sets a custom output destination for logs (useful for unit testing).
func WithWriter(w io.Writer) LoggerOption {
	return func(o *loggerOptions) {
		if w != nil {
			o.writer = w
		}
	}
}

// StructuredLogger returns a Gin middleware that outputs structured JSON logs for each HTTP request.
func StructuredLogger(opts ...LoggerOption) gin.HandlerFunc {
	cfg := &loggerOptions{
		writer: os.Stdout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *gin.Context) {
		start := time.Now()

		// 1. Extract or generate Request ID
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		} else if _, err := uuid.Parse(reqID); err != nil {
			reqID = uuid.New().String()
		}

		// Inject request_id into response headers and contexts
		c.Header("X-Request-ID", reqID)
		c.Set("request_id", reqID)
		if c.Request != nil {
			c.Request = c.Request.WithContext(ctxmeta.WithRequestID(c.Request.Context(), reqID))
		}

		// 2. Process request
		c.Next()

		// 3. Compute latency & level
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		var level string
		switch {
		case statusCode >= 500:
			level = "ERROR"
		case statusCode >= 400:
			level = "WARN"
		default:
			level = "INFO"
		}

		entry := LogEntry{
			Level:      level,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			RequestID:  reqID,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: statusCode,
			Latency:    latency.String(),
			LatencyMs:  latency.Milliseconds(),
			ClientIP:   c.ClientIP(),
		}

		data, err := json.Marshal(entry)
		if err == nil {
			data = append(data, '\n')
			if n, writeErr := cfg.writer.Write(data); writeErr != nil {
				log.Printf("failed to write structured log (bytes: %d): %v", n, writeErr)
			}
		}
	}
}
