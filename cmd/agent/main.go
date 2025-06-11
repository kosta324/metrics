package main

import (
	"flag"
	"fmt"
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

		case <-reportTicker.C:
			for name, val := range metrics {
				sendMetric("gauge", name, fmt.Sprintf("%f", val))
			}
			sendMetric("counter", "PollCount", strconv.FormatInt(pollCount, 10))
		}
	}
}

func sendMetric(metricType, name, value string) {
	url := fmt.Sprintf("http://%s/update/%s/%s/%s", *serverAddress, metricType, name, value)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error sending metric %s: %v\n", name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("server returned non-OK for %s: %d\n", name, resp.StatusCode)
	}
}
