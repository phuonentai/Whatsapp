package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// gzipResponseWriter wraps gin.ResponseWriter to transparently gzip the body.
type gzipResponseWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.writer.Write([]byte(s))
}

// excludedCompressionPaths are endpoints that must never be compressed:
// the SSE chat stream (gzip buffering would break token streaming) and
// /metrics (scrape requests already negotiate encoding explicitly).
var excludedCompressionPaths = map[string]struct{}{
	"/api/example_cognitive/chat": {},
	"/metrics":                    {},
}

// Compression compresses responses with gzip when the client supports it.
// Streaming and excluded paths are never compressed.
func Compression() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if _, excluded := excludedCompressionPaths[path]; excluded {
			c.Next()
			return
		}
		if c.Writer.Header().Get("Content-Encoding") != "" {
			c.Next()
			return
		}
		if !acceptsGzip(c.Request.Header.Get("Accept-Encoding")) {
			c.Next()
			return
		}

		gz := gzip.NewWriter(c.Writer)
		c.Writer = &gzipResponseWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}
		c.Writer.Header().Set("Content-Encoding", "gzip")
		c.Writer.Header().Del("Content-Length")

		defer func() {
			_ = gz.Close()
		}()

		c.Next()
	}
}

func acceptsGzip(acceptEncoding string) bool {
	for _, part := range strings.Split(acceptEncoding, ",") {
		part = strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			return true
		}
	}
	return false
}

// verify gzipResponseWriter satisfies http.Flusher so SSE writers still flush.
var _ http.Flusher = (*gzipResponseWriter)(nil)

func (w *gzipResponseWriter) Flush() {
	w.writer.Flush()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
