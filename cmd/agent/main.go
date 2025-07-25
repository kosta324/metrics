package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kosta324/metrics.git/internal/models"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

var (
	serverAddress  = flag.String("a", "localhost:8080", "HTTP server address")
	pollInterval   = flag.Int("p", 2, "Poll interval (seconds)")
	reportInterval = flag.Int("r", 10, "Report interval (seconds)")
	log            *zap.SugaredLogger
)

var pollCount int64

func initLogger() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	log = logger.Sugar()
}

func main() {
	initLogger()
	defer log.Sync()

	flag.Parse()
	if len(flag.Args()) > 0 {
		log.Fatalf("unknown arguments: %v\n", flag.Args())
		return
	}

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		*serverAddress = envAddr
	}

	if envPoll, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if val, err := strconv.Atoi(envPoll); err == nil {
			*pollInterval = val
		}
	}

	if envReport, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if val, err := strconv.Atoi(envReport); err == nil {
			*reportInterval = val
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	pollTicker := time.NewTicker(time.Duration(*pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(*reportInterval) * time.Second)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	metrics := make(map[string]float64)

	go func() {
		for {
			select {
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
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-reportTicker.C:
				var metricsBatch []models.Metrics
				for name, val := range metrics {
					m := models.Metrics{
						ID:    name,
						MType: "gauge",
						Value: &val,
					}
					metricsBatch = append(metricsBatch, m)
				}
				m := models.Metrics{
					ID:    "PollCount",
					MType: "counter",
					Delta: &pollCount,
				}
				metricsBatch = append(metricsBatch, m)
				if err := sendMetricsBatch(ctx, metricsBatch); err != nil {
					log.Errorf("failed to send metrics batch (including PollCount): %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-stop
	log.Info("Agent shutting down...")
	cancel()
}

func sendMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(body)
	if err != nil {
		return fmt.Errorf("error compressing: %v", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("error closing gzip: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+*serverAddress+"/updates/", &buf)
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
