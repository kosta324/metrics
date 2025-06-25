package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMetricJSON(t *testing.T) {
	t.Run("send gauge metric", func(t *testing.T) {
		value := 123.456
		metric := Metrics{
			ID:    "TestGauge",
			MType: "gauge",
			Value: &value,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/update/", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var got Metrics
			err = json.Unmarshal(body, &got)
			require.NoError(t, err)

			assert.Equal(t, metric.ID, got.ID)
			assert.Equal(t, metric.MType, got.MType)
			assert.Equal(t, *metric.Value, *got.Value)
		}))
		defer server.Close()

		*serverAddress = server.URL[len("http://"):]
		sendMetricJSON(metric)
	})

	t.Run("send counter metric", func(t *testing.T) {
		count := int64(5)
		metric := Metrics{
			ID:    "TestCounter",
			MType: "counter",
			Delta: &count,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/update/", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var got Metrics
			err = json.Unmarshal(body, &got)
			require.NoError(t, err)
			assert.Equal(t, metric.ID, got.ID)
			assert.Equal(t, metric.MType, got.MType)
			assert.Equal(t, *metric.Delta, *got.Delta)
		}))
		defer server.Close()

		*serverAddress = server.URL[len("http://"):]
		sendMetricJSON(metric)
	})
}
