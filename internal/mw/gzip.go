package mw

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Gzip поддерживает:
// - распаковку request body при Content-Encoding: gzip
// - сжатие response при Accept-Encoding: gzip
// Сжатие применяется только для Content-Type: application/json и text/html.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request decompression
		if isGzipEncoding(r.Header.Get("Content-Encoding")) {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			r.Body = &gzipReadCloser{Reader: gr, body: r.Body}
			r.Header.Del("Content-Encoding")
		}

		// response compression
		if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		gw := &gzipResponseWriter{ResponseWriter: w}
		defer gw.Close()

		next.ServeHTTP(gw, r)
	})
}

type gzipReadCloser struct {
	*gzip.Reader
	body io.Closer
}

func (g *gzipReadCloser) Close() error {
	_ = g.Reader.Close()
	return g.body.Close()
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
	enabled     bool
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	if g.wroteHeader {
		g.ResponseWriter.WriteHeader(statusCode)
		return
	}
	g.wroteHeader = true

	// Решение о сжатии принимаем по Content-Type, который выставил хендлер.
	ct := g.Header().Get("Content-Type")
	if isCompressibleContentType(ct) {
		g.enabled = true
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Add("Vary", "Accept-Encoding")
		g.Header().Del("Content-Length")
		g.gz = gzip.NewWriter(g.ResponseWriter)
	}

	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	if !g.wroteHeader {
		// если заголовки ещё не отправлены — выставляем 200 и решаем по Content-Type
		g.WriteHeader(http.StatusOK)
	}
	if !g.enabled {
		return g.ResponseWriter.Write(p)
	}
	return g.gz.Write(p)
}

func (g *gzipResponseWriter) Flush() {
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		if g.enabled && g.gz != nil {
			_ = g.gz.Flush()
		}
		f.Flush()
	}
}

func (g *gzipResponseWriter) Close() error {
	if g.enabled && g.gz != nil {
		return g.gz.Close()
	}
	return nil
}

func acceptsGzip(v string) bool {
	// достаточно простая проверка на наличие gzip в списке кодировок
	return strings.Contains(strings.ToLower(v), "gzip")
}

func isGzipEncoding(v string) bool {
	return strings.Contains(strings.ToLower(v), "gzip")
}

func isCompressibleContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	// допускаем параметры типа charset
	return strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/html")
}

