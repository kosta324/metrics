package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendMetricJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var m Metrics
		err = json.Unmarshal(body, &m)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(m)
	}))
	defer ts.Close()

	serverAddress = &ts.URL

	t.Run("send gauge metric", func(t *testing.T) {
		val := 123.456
		metric := Metrics{
			ID:    "TestGauge",
			MType: "gauge",
			Value: &val,
		}
		sendMetricJSON(metric)
	})

	t.Run("send counter metric", func(t *testing.T) {
		delta := int64(42)
		metric := Metrics{
			ID:    "TestCounter",
			MType: "counter",
			Delta: &delta,
		}
		sendMetricJSON(metric)
	})
}
