package middleware

import (
	"compress/gzip"
	"io"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Compression levels
const (
	DefaultCompression = gzip.DefaultCompression
	BestCompression    = gzip.BestCompression
	BestSpeed          = gzip.BestSpeed
)

// gzipWriterPool reuses gzip writers to reduce allocations
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		gz, _ := gzip.NewWriterLevel(io.Discard, DefaultCompression)
		return gz
	},
}

// gzipWriter wraps gin.ResponseWriter with gzip compression
type gzipWriter struct {
	gin.ResponseWriter
	gw *gzip.Writer
}

// Write writes data through gzip compression
func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.gw.Write(data)
}

// WriteString writes string data through gzip compression
func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.gw.Write([]byte(s))
}

// shouldCompress determines if response should be compressed
func shouldCompress(c *gin.Context) bool {
	// Check Accept-Encoding header
	if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}

	// Check content type - only compress text-based content
	contentType := c.Writer.Header().Get("Content-Type")
	if contentType == "" {
		contentType = c.ContentType()
	}

	compressibleTypes := []string{
		"application/json",
		"application/javascript",
		"text/html",
		"text/css",
		"text/plain",
		"text/xml",
		"application/xml",
		"application/x-javascript",
	}

	for _, ct := range compressibleTypes {
		if strings.Contains(contentType, ct) {
			return true
		}
	}

	return false
}

// isStreamingResponse checks if response is streaming (should not be compressed)
func isStreamingResponse(c *gin.Context) bool {
	// Check for SSE or streaming responses
	contentType := c.Writer.Header().Get("Content-Type")
	return strings.Contains(contentType, "text/event-stream") ||
		strings.Contains(contentType, "application/octet-stream")
}

// CompressionMiddleware provides gzip compression for API responses
func CompressionMiddleware() gin.HandlerFunc {
	return CompressionMiddlewareWithLevel(DefaultCompression)
}

// CompressionMiddlewareWithLevel provides configurable gzip compression
func CompressionMiddlewareWithLevel(level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip compression for certain conditions
		if c.Request.Method == "HEAD" ||
			c.Request.Header.Get("Connection") == "Upgrade" ||
			isStreamingResponse(c) ||
			!shouldCompress(c) {
			c.Next()
			return
		}

		// Get gzip writer from pool
		gz := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gz)

		// Reset writer with response writer and compression level
		gz.Reset(c.Writer)
		if level != DefaultCompression {
			// If custom level, create new writer (pool optimization for default level only)
			if customGz, err := gzip.NewWriterLevel(c.Writer, level); err == nil {
				gz = customGz
			}
		}
		defer gz.Close()

		// Set compression headers
		c.Writer.Header().Set("Content-Encoding", "gzip")
		c.Writer.Header().Set("Vary", "Accept-Encoding")
		c.Writer.Header().Del("Content-Length") // Let gzip determine the length

		// Wrap response writer
		c.Writer = &gzipWriter{
			ResponseWriter: c.Writer,
			gw:            gz,
		}

		c.Next()
	}
}

// NoCompressionMiddleware explicitly disables compression for specific routes
func NoCompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-No-Compression", "1")
		c.Next()
	}
}