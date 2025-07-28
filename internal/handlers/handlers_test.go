package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/models"
	"github.com/kosta324/metrics.git/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateMetricJSON(t *testing.T) {
	type want struct {
		code int
	}
	tests := []struct {
		name  string
		input models.Metrics
		want  want
	}{
		{
			name: "update gauge via JSON",
			input: models.Metrics{
				ID:    "TestGauge",
				MType: "gauge",
				Value: func() *float64 { v := 123.456; return &v }(),
			},
			want: want{code: http.StatusOK},
		},
		{
			name: "update counter via JSON",
			input: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
				Delta: func() *int64 { d := int64(42); return &d }(),
			},
			want: want{code: http.StatusOK},
		},
	}

	repo := storage.NewMemStorage()
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create logger")
	defer logger.Sync()
	log := logger.Sugar()
	h := NewHandler(repo, log)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)
		})
	}
}

func TestGetMetricJSON(t *testing.T) {
	type want struct {
		code  int
		value float64
	}
	tests := []struct {
		name  string
		input models.Metrics
		want  want
	}{
		{
			name: "get gauge via JSON",
			input: models.Metrics{
				ID:    "GaugeTwoDecimals",
				MType: "gauge",
			},
			want: want{
				code:  http.StatusOK,
				value: 603057.87,
			},
		},
		{
			name: "get existing counter",
			input: models.Metrics{
				ID:    "PollCount",
				MType: "counter",
			},
			want: want{
				code:  http.StatusOK,
				value: 7,
			},
		},
		{
			name: "get non-existing metric",
			input: models.Metrics{
				ID:    "UnknownMetric",
				MType: "gauge",
			},
			want: want{
				code:  http.StatusNotFound,
				value: 0,
			},
		},
	}

	repo := storage.NewMemStorage()
	_ = repo.Add("gauge", "GaugeTwoDecimals", "603057.87")
	_ = repo.Add("counter", "PollCount", "7")
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create logger")
	defer logger.Sync()
	log := logger.Sugar()
	h := NewHandler(repo, log)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.code == http.StatusOK {
				var got models.Metrics
				err := json.NewDecoder(res.Body).Decode(&got)
				require.NoError(t, err)
				assert.Equal(t, tt.input.ID, got.ID)
				assert.Equal(t, tt.input.MType, got.MType)
				if got.MType == "gauge" {
					assert.NotNil(t, got.Value)
					assert.InDelta(t, tt.want.value, *got.Value, 0.0001)
				} else if got.MType == "counter" {
					assert.NotNil(t, got.Delta)
					assert.Equal(t, int64(tt.want.value), *got.Delta)
				}
			}
		})
	}
}

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
	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create logger")
	defer logger.Sync()
	log := logger.Sugar()
	h := NewHandler(repo, log)

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

func setupRouterWithTestData(t *testing.T) http.Handler {
	repo := storage.NewMemStorage()
	_ = repo.Add("gauge", "GaugeTwoDecimals", "603057.87")
	_ = repo.Add("gauge", "GaugeThreeDecimals", "550386.837")
	_ = repo.Add("counter", "PollCount", "7")

	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create logger")
	defer logger.Sync()
	log := logger.Sugar()
	h := NewHandler(repo, log)

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

	router := setupRouterWithTestData(t)

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
