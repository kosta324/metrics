package main

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/kosta324/metrics.git/internal/storage"
)

type Client struct {
	PollInterval   int
	ReportInterval int
	HostAddress    string
	Repo           storage.Repositories
}

func (client *Client) Run() {
	start := time.Now()
	var stat runtime.MemStats
	var interval int
	for {
		client.poll(&stat)
		client.Repo.Increase()
		time.Sleep(time.Duration(client.PollInterval * int(time.Second)))
		interval = int(time.Until(start).Abs().Seconds())
		if interval >= client.ReportInterval {
			client.report()
			start = time.Now()
		}
	}
}

func (client *Client) poll(stat *runtime.MemStats) {
	runtime.ReadMemStats(stat)
	client.Repo.Add(stat)
}

func (client *Client) report() {
	fmt.Printf("Metric = %v \n", client.Repo)
	const counterType = "text/plain"
	reader := bytes.NewReader([]byte(""))
	for name, val := range client.Repo.GetGauge() {
		_, err := http.Post(fmt.Sprintf("%s/update/%s/%s/%f", client.HostAddress, "gauge", name, val), counterType, reader)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
	for name, val := range client.Repo.GetCounter() {
		_, err := http.Post(fmt.Sprintf("%s/update/%s/%s/%d", client.HostAddress, "counter", name, val), counterType, reader)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}

func main() {
	const pollInterval = 2
	const reportInterval = 10
	const hostAddress = "http://localhost:8080"
	repo := storage.InitStorage()
	client := Client{
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		HostAddress:    hostAddress,
		Repo:           &repo,
	}
	client.Run()
}
