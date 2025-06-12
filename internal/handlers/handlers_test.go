package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/storage"
	"github.com/stretchr/testify/assert"
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

	repo := storage.NewMemStorage()
	h := NewHandler(repo)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.contentType != "" {
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}

func setupRouterWithTestData() http.Handler {
	repo := storage.NewMemStorage()
	_ = repo.Add("gauge", "GaugeTwoDecimals", "603057.87")
	_ = repo.Add("gauge", "GaugeThreeDecimals", "550386.837")
	_ = repo.Add("counter", "PollCount", "7")

	h := NewHandler(repo)

	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func TestGetMetrics(t *testing.T) {
	type want struct {
		code     int
		body     string
		contains string
	}

	tests := []struct {
		name string
		url  string
		want want
	}{
		{
			name: "GET existing gauge with 2 decimals",
			url:  "/value/gauge/GaugeTwoDecimals",
			want: want{
				code: http.StatusOK,
				body: "603057.87\n",
			},
		},
		{
			name: "GET existing gauge with 3 decimals",
			url:  "/value/gauge/GaugeThreeDecimals",
			want: want{
				code: http.StatusOK,
				body: "550386.837\n",
			},
		},
		{
			name: "GET existing counter",
			url:  "/value/counter/PollCount",
			want: want{
				code: http.StatusOK,
				body: "7\n",
			},
		},
		{
			name: "GET non-existing metric",
			url:  "/value/gauge/UnknownMetric",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name: "GET / returns HTML with all metrics formatted",
			url:  "/",
			want: want{
				code:     http.StatusOK,
				contains: "<li><b>GaugeTwoDecimals</b>: 603057.87</li>",
			},
		},
	}

	router := setupRouterWithTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.body != "" {
				assert.Equal(t, tt.want.body, string(body))
			}
			if tt.want.contains != "" {
				assert.Contains(t, string(body), tt.want.contains)
				assert.NotContains(t, string(body), "603057.870")
			}
		})
	}
}
