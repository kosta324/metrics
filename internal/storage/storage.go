package storage

import (
	"errors"
	"fmt"
	"strconv"
)

type Repository interface {
	Add(metricType, name, value string) error
	Get(metricType, name string) (string, error)
	GetAll() map[string]string
}

type gauge float64
type counter int64

type MemStorage struct {
	Gauges   map[string]gauge
	Counters map[string]counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Gauges:   make(map[string]gauge),
		Counters: make(map[string]counter),
	}
}

func (ms *MemStorage) Add(metricType, name, value string) error {
	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		ms.Gauges[name] = gauge(val)
	case "counter":
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		ms.Counters[name] += counter(val)
	default:
		return errors.New("unsupported metric type")
	}
	return nil
}

func (ms *MemStorage) Get(metricType, name string) (string, error) {
	switch metricType {
	case "gauge":
		val, ok := ms.Gauges[name]
		if !ok {
			return "", errors.New("not found")
		}
		return strconv.FormatFloat(float64(val), 'f', -1, 64), nil
	case "counter":
		val, ok := ms.Counters[name]
		if !ok {
			return "", errors.New("not found")
		}
		return fmt.Sprintf("%d", val), nil
	default:
		return "", errors.New("unsupported metric type")
	}
}

func (ms *MemStorage) GetAll() map[string]string {
	result := make(map[string]string)

	for k, v := range ms.Gauges {
		result[k] = strconv.FormatFloat(float64(v), 'f', -1, 64)
	}
	for k, v := range ms.Counters {
		result[k] = fmt.Sprintf("%d", v)
	}
	return result
}
