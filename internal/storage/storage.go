package storage

import (
	"math/rand"
	"runtime"
)

type Repositories interface {
	Add(stat *runtime.MemStats) error
	Increase()
	GetGauge() map[string]gauge
	GetCounter() map[string]counter
}

type gauge float64
type counter int64

type memStorage struct {
	Gauge   map[string]gauge
	Counter map[string]counter
}

func InitStorage() memStorage {
	return memStorage{
		Gauge:   make(map[string]gauge),
		Counter: make(map[string]counter),
	}
}

func (metric *memStorage) Add(stat *runtime.MemStats) error {
	metric.Gauge["Alloc"] = gauge(stat.Alloc)
	metric.Gauge["BuckHashSys"] = gauge(stat.BuckHashSys)
	metric.Gauge["Frees"] = gauge(stat.Frees)
	metric.Gauge["GCCPUFraction"] = gauge(stat.GCCPUFraction)
	metric.Gauge["GCSys"] = gauge(stat.GCSys)
	metric.Gauge["HeapAlloc"] = gauge(stat.HeapAlloc)
	metric.Gauge["HeapIdle"] = gauge(stat.HeapIdle)
	metric.Gauge["HeapInuse"] = gauge(stat.HeapInuse)
	metric.Gauge["HeapObjects"] = gauge(stat.HeapObjects)
	metric.Gauge["HeapReleased"] = gauge(stat.HeapReleased)
	metric.Gauge["HeapSys"] = gauge(stat.HeapSys)
	metric.Gauge["LastGC"] = gauge(stat.LastGC)
	metric.Gauge["Lookups"] = gauge(stat.Lookups)
	metric.Gauge["MCacheInuse"] = gauge(stat.MCacheInuse)
	metric.Gauge["MCacheSys"] = gauge(stat.MCacheSys)
	metric.Gauge["MSpanInuse"] = gauge(stat.MSpanInuse)
	metric.Gauge["MSpanSys"] = gauge(stat.MSpanSys)
	metric.Gauge["Mallocs"] = gauge(stat.Mallocs)
	metric.Gauge["NextGC"] = gauge(stat.NextGC)
	metric.Gauge["NumForcedGC"] = gauge(stat.NumForcedGC)
	metric.Gauge["NumGC"] = gauge(stat.NumGC)
	metric.Gauge["OtherSys"] = gauge(stat.OtherSys)
	metric.Gauge["PauseTotalNs"] = gauge(stat.PauseTotalNs)
	metric.Gauge["StackInuse"] = gauge(stat.StackInuse)
	metric.Gauge["StackSys"] = gauge(stat.StackSys)
	metric.Gauge["Sys"] = gauge(stat.Sys)
	metric.Gauge["TotalAlloc"] = gauge(stat.TotalAlloc)
	metric.Gauge["RandomValue"] = gauge(rand.Int())
	return nil
}

func (metric *memStorage) Increase() {
	metric.Counter["PollCount"]++
}

func (metric *memStorage) GetGauge() map[string]gauge {
	return metric.Gauge
}

func (metric *memStorage) GetCounter() map[string]counter {
	return metric.Counter
}
