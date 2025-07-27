package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kosta324/metrics.git/internal/agent"
	"github.com/kosta324/metrics.git/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSendMetricsBatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "/updates/", r.URL.Path)

		gr, err := gzip.NewReader(r.Body)
		require.NoError(t, err)
		defer gr.Close()

		body, err := io.ReadAll(gr)
		require.NoError(t, err)

		var batch []models.Metrics
		err = json.Unmarshal(body, &batch)
		require.NoError(t, err)

		assert.Greater(t, len(batch), 0)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(batch)
	}))
	defer ts.Close()

	logger, err := zap.NewDevelopment()
	require.NoError(t, err, "failed to create logger")
	defer logger.Sync()
	log := logger.Sugar()

	addr := ts.URL[len("http://"):]

	t.Run("send gauge metric batch", func(t *testing.T) {
		val := 123.456
		metric := models.Metrics{
			ID:    "TestGauge",
			MType: "gauge",
			Value: &val,
		}
		err := agent.SendMetricsBatch(context.Background(), addr, []models.Metrics{metric}, log)
		assert.NoError(t, err)
	})

	t.Run("send counter metric batch", func(t *testing.T) {
		delta := int64(42)
		metric := models.Metrics{
			ID:    "TestCounter",
			MType: "counter",
			Delta: &delta,
		}
		err := agent.SendMetricsBatch(context.Background(), addr, []models.Metrics{metric}, log)
		assert.NoError(t, err)
	})

	t.Run("send empty metric batch", func(t *testing.T) {
		err := agent.SendMetricsBatch(context.Background(), addr, []models.Metrics{}, log)
		assert.NoError(t, err)
	})
}
