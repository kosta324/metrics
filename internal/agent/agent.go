package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kosta324/metrics.git/internal/models"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Config struct {
	ServerAddress  string
	PollInterval   int
	ReportInterval int
}

var pollCount int64

func (c *Config) ApplyEnv() {
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		c.ServerAddress = v
	}
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if i, err := strconv.Atoi(v); err == nil {
			c.PollInterval = i
		}
	}
	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if i, err := strconv.Atoi(v); err == nil {
			c.ReportInterval = i
		}
	}
}

func Run(ctx context.Context, cfg Config, log *zap.SugaredLogger) error {
	pollTicker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	metrics := make(map[string]float64)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				pollCount++
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				metrics["Alloc"] = float64(m.Alloc)
				metrics["BuckHashSys"] = float64(m.BuckHashSys)
				metrics["Frees"] = float64(m.Frees)
				metrics["GCCPUFraction"] = m.GCCPUFraction
				metrics["GCSys"] = float64(m.GCSys)
				metrics["HeapAlloc"] = float64(m.HeapAlloc)
				metrics["HeapIdle"] = float64(m.HeapIdle)
				metrics["HeapInuse"] = float64(m.HeapInuse)
				metrics["HeapObjects"] = float64(m.HeapObjects)
				metrics["HeapReleased"] = float64(m.HeapReleased)
				metrics["HeapSys"] = float64(m.HeapSys)
				metrics["LastGC"] = float64(m.LastGC)
				metrics["Lookups"] = float64(m.Lookups)
				metrics["MCacheInuse"] = float64(m.MCacheInuse)
				metrics["MCacheSys"] = float64(m.MCacheSys)
				metrics["MSpanInuse"] = float64(m.MSpanInuse)
				metrics["MSpanSys"] = float64(m.MSpanSys)
				metrics["Mallocs"] = float64(m.Mallocs)
				metrics["NextGC"] = float64(m.NextGC)
				metrics["NumForcedGC"] = float64(m.NumForcedGC)
				metrics["NumGC"] = float64(m.NumGC)
				metrics["OtherSys"] = float64(m.OtherSys)
				metrics["PauseTotalNs"] = float64(m.PauseTotalNs)
				metrics["StackInuse"] = float64(m.StackInuse)
				metrics["StackSys"] = float64(m.StackSys)
				metrics["Sys"] = float64(m.Sys)
				metrics["TotalAlloc"] = float64(m.TotalAlloc)
				metrics["RandomValue"] = rand.Float64()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info("agent stopped")
			return nil
		case <-reportTicker.C:
			var batch []models.Metrics
			for name, val := range metrics {
				m := models.Metrics{
					ID:    name,
					MType: "gauge",
					Value: &val,
				}
				batch = append(batch, m)
			}
			batch = append(batch, models.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: &pollCount,
			})
			if err := SendMetricsBatch(ctx, cfg.ServerAddress, batch, log); err != nil {
				log.Errorf("failed to send metrics batch: %v", err)
			}
		}
	}
}

func SendMetricsBatch(ctx context.Context, server string, metrics []models.Metrics, log *zap.SugaredLogger) error {
	if len(metrics) == 0 {
		return nil
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error creating request body: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err = gz.Write(body); err != nil {
		return fmt.Errorf("error compressing: %v", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("error closing gzip: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+server+"/updates/", &buf)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending metrics batch: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK for batch: %d", resp.StatusCode)
	}

	return nil
}
