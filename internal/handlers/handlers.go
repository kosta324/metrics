package handlers

import (
	"net/http"
	"strconv"
)

type Repositories interface {
	Add(mType string, name string, value float64) error
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

func (metric *memStorage) Add(mType string, name string, value float64) error {
	switch mType {
	case "gauge":
		metric.Gauge[name] = gauge(value)
	case "counter":
		metric.Counter[name] += counter(value)
	}

	return nil
}

func Post(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		next.ServeHTTP(res, req)
	})
}

type NewHandler struct {
	Repo Repositories
}

func (h *NewHandler) update(res http.ResponseWriter, req *http.Request) {
	typeMetric := req.PathValue("type")
	nameMetric := req.PathValue("name")
	valMetric := req.PathValue("val")
	var val float64

	if nameMetric == "" {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	if typeMetric != "gauge" && typeMetric != "counter" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	val, err := strconv.ParseFloat(valMetric, 64)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	h.Repo.Add(typeMetric, nameMetric, val)

	res.WriteHeader(http.StatusOK)
}

func (h *NewHandler) Handle(mux *http.ServeMux) {
	mux.Handle("/update/", Post(http.HandlerFunc(h.update)))
}
