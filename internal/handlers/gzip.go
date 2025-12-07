package handlers

import (
	"compress/gzip"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// compressWriter implements gin.ResponseWriter
type compressWriter struct {
	gin.ResponseWriter
	zw *gzip.Writer
}

// Write compresses data before writing to the response
func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// WriteString writes string data with compression
func (c *compressWriter) WriteString(s string) (int, error) {
	return c.zw.Write([]byte(s))
}

// Close closes the gzip writer
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// Pool for gzip writers to reuse them
var gzipPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
		return w
	},
}

// gzipMiddleware returns a gin middleware that enables gzip compression
func (h *Handlers) gzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Check if client supports gzip compression for response
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		// Get gzip writer from pool
		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)
		gz.Reset(c.Writer)

		// Wrap the response writer
		gzWriter := &compressWriter{
			ResponseWriter: c.Writer,
			zw:             gz,
		}
		c.Writer = gzWriter

		// Set headers
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		defer func() {
			gz.Close()
		}()

		c.Next()
	}
}
