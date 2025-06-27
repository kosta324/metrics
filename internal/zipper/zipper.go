package zipper

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	body   bytes.Buffer
	status int
	wrote  bool
}

func (w *wrappedResponseWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
	}
}

func (w *wrappedResponseWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "failed to decompress request", http.StatusBadRequest)
				return
			}
			defer gr.Close()
			r.Body = io.NopCloser(gr)
		}

		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		if !acceptsGzip {
			next.ServeHTTP(w, r)
			return
		}

		wrw := &wrappedResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrw, r)

		contentType := wrw.Header().Get("Content-Type")
		if !(strings.HasPrefix(contentType, "application/json") || strings.HasPrefix(contentType, "text/html")) {
			w.WriteHeader(wrw.status)
			w.Write(wrw.body.Bytes())
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(wrw.status)

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, "failed to init gzip writer", http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		gz.Write(wrw.body.Bytes())
	})
}
