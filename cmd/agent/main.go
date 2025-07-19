package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kosta324/metrics.git/internal/models"
	"io"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

var (
	serverAddress  = flag.String("a", "localhost:8080", "HTTP server address")
	pollInterval   = flag.Int("p", 2, "Poll interval (seconds)")
	reportInterval = flag.Int("r", 10, "Report interval (seconds)")
)

var pollCount int64

func main() {
	flag.Parse()
	if len(flag.Args()) > 0 {
		fmt.Printf("unknown arguments: %v\n", flag.Args())
		return
	}

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*serverAddress = envAddr
	}

	if envPoll := os.Getenv("POLL_INTERVAL"); envPoll != "" {
		if val, err := strconv.Atoi(envPoll); err == nil {
			*pollInterval = val
		}
	}

	if envReport := os.Getenv("REPORT_INTERVAL"); envReport != "" {
		if val, err := strconv.Atoi(envReport); err == nil {
			*reportInterval = val
		}
	}

	pollTicker := time.NewTicker(time.Duration(*pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(*reportInterval) * time.Second)

	metrics := make(map[string]float64)

	go func() {
		for range pollTicker.C {
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
	}()

	for range reportTicker.C {
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
		sendMetricsBatch(metricsBatch)
	}
}

func sendMetricsBatch(metrics []models.Metrics) {
	if len(metrics) == 0 {
		return
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(body)
	if err != nil {
		fmt.Printf("error compressing: %v\n", err)
		return
	}
	gz.Close()

	req, err := http.NewRequest(http.MethodPost, "http://"+*serverAddress+"/updates/", &buf)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error sending metrics batch: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("server returned non-OK for batch: %d\n", resp.StatusCode)
	}
}

func sendMetricJSON(m models.Metrics) {
	body, err := json.Marshal(m)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(body)
	if err != nil {
		fmt.Printf("error compressing request: %v\n", err)
		return
	}
	gz.Close()

	url := "http://" + *serverAddress + "/update/"
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{}

	var resp *http.Response
	retries := []time.Duration{0, 1 * time.Second, 3 * time.Second, 5 * time.Second}
	for attempt, delay := range retries {
		if attempt > 0 {
			time.Sleep(delay)
		}
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			break
		}
		if err != nil {
			fmt.Printf("attempt %d: error sending metric %s: %v\n", attempt+1, m.ID, err)
		} else if resp != nil {
			fmt.Printf("attempt %d: server returned status %d for %s\n", attempt+1, resp.StatusCode, m.ID)
		}
	}
	if err != nil {
		fmt.Printf("final error sending metric %s: %v\n", m.ID, err)
		return
	}

	var reader io.Reader = resp.Body

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			fmt.Printf("error decompressing response: %v\n", err)
			return
		}
		defer gzr.Close()
		reader = gzr
	}

	respBody, err := io.ReadAll(reader)
	if err != nil {
		fmt.Printf("error reading response body: %v\n", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("server returned non-OK for %s: %d, body: %s\n", m.ID, resp.StatusCode, string(respBody))
	}
}
