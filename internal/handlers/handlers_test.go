package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kosta324/metrics.git/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainPage(t *testing.T) {
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
				code:        405,
				contentType: "text/plain; charset=utf-8",
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

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			
			if tt.name == "missing metric name" {
				// вызывать напрямую без ServeMux
				h.MainPage(w, req)
			} else {
				mux := http.NewServeMux()
				mux.HandleFunc("/update/", h.MainPage)
				mux.ServeHTTP(w, req)
			}

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
