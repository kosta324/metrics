package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMetric(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}

	tests := []struct {
		name   string
		method string
		url    string
		want   want
	}{
		{
			name:   "valid gauge metric",
			method: http.MethodPost,
			url:    "/update/gauge/HeapAlloc/123.45",
			want: want{
				code: 200,
			},
		},
		{
			name:   "valid counter metric",
			method: http.MethodPost,
			url:    "/update/counter/PollCount/42",
			want: want{
				code: 200,
			},
		},
		{
			name:   "invalid method",
			method: http.MethodGet,
			url:    "/update/counter/MetricName/1",
			want: want{
				code: 405,
			},
		},
		{
			name:   "invalid metric type",
			method: http.MethodPost,
			url:    "/update/invalid/Metric/123",
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "invalid gauge value",
			method: http.MethodPost,
			url:    "/update/gauge/SomeMetric/abc",
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "invalid counter value",
			method: http.MethodPost,
			url:    "/update/counter/SomeMetric/NaN",
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "missing metric name",
			method: http.MethodPost,
			url:    "/update/counter//123",
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := storage.NewMemStorage()
			h := NewHandler(repo)

			r := chi.NewRouter()
			h.RegisterRoutes(r)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.contentType != "" {
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			}

			_, err := io.ReadAll(res.Body)
			require.NoError(t, err)
		})
	}
}

func TestGetAndListMetrics(t *testing.T) {
	repo := storage.NewMemStorage()
	_ = repo.Add("gauge", "HeapAlloc", "123.45")
	_ = repo.Add("counter", "PollCount", "7")

	h := NewHandler(repo)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	t.Run("GET existing gauge metric", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/gauge/HeapAlloc", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		body, _ := io.ReadAll(res.Body)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(body), "123.45")
	})

	t.Run("GET existing counter metric", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		body, _ := io.ReadAll(res.Body)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(body), "7")
	})

	t.Run("GET non-existent metric returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/gauge/Unknown", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("GET / returns HTML with all metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		body, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, res.Header.Get("Content-Type"), "text/html")
		assert.Contains(t, string(body), "HeapAlloc")
		assert.Contains(t, string(body), "PollCount")
	})
}
