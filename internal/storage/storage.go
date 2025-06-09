package storage

import (
	"errors"
	"strconv"
)

type Repository interface {
	Add(metricType string, name string, value string) error
}

type gauge float64
type counter int64

type MemStorage struct {
	Gauge   map[string]gauge
	Counter map[string]counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Gauge: make(map[string]gauge),
		Counter: make(map[string]counter),
	}
}

func (ms *MemStorage) Add(metricType string, name string, value string) error {
	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		ms.Gauge[name] = gauge(val)
	case "counter":
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		ms.Counter[name] += counter(val)
	default:
		return errors.New("unsupported metric type")
	}
	return nil
}
