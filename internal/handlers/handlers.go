package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/models"
	"github.com/kosta324/metrics.git/internal/storage"
	"go.uber.org/zap"
)

type Handler struct {
	Repo   storage.Repository
	logger *zap.SugaredLogger
}

func NewHandler(repo storage.Repository, log *zap.SugaredLogger) *Handler {
	return &Handler{
		Repo:   repo,
		logger: log,
	}
}

func (h *Handler) PingDB(w http.ResponseWriter, r *http.Request) {
	if err := h.Repo.Ping(); err != nil {
		h.logger.Errorf("database ping failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/update/", h.UpdateMetricJSON)
	r.Post("/updates/", h.UpdateMetricsBatch)
	r.Post("/value/", h.GetMetricJSON)
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Get("/", h.ListMetrics)
	r.Get("/ping", h.PingDB)
}

func (h *Handler) UpdateMetricsBatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metrics []models.Metrics
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &metrics); err != nil {
		http.Error(w, "invalid JSON array", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "empty metrics batch", http.StatusBadRequest)
		return
	}

	if sqlRepo, ok := h.Repo.(*storage.SQLRepo); ok {
		if err := sqlRepo.AddBatch(metrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		for _, m := range metrics {
			switch m.MType {
			case "gauge":
				if m.Value == nil {
					http.Error(w, "missing gauge value", http.StatusBadRequest)
					return
				}
				if err := h.Repo.Add("gauge", m.ID, strconv.FormatFloat(*m.Value, 'f', -1, 64)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case "counter":
				if m.Delta == nil {
					http.Error(w, "missing counter delta", http.StatusBadRequest)
					return
				}
				if err := h.Repo.Add("counter", m.ID, strconv.FormatInt(*m.Delta, 10)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				http.Error(w, "unknown metric type", http.StatusNotImplemented)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m models.Metrics
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &m)
	if err != nil || m.ID == "" || m.MType == "" {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	switch m.MType {
	case "gauge":
		if m.Value == nil {
			http.Error(w, "missing gauge value", http.StatusBadRequest)
			return
		}
		err = h.Repo.Add("gauge", m.ID, strconv.FormatFloat(*m.Value, 'f', -1, 64))
	case "counter":
		if m.Delta == nil {
			http.Error(w, "missing counter delta", http.StatusBadRequest)
			return
		}
		err = h.Repo.Add("counter", m.ID, strconv.FormatInt(*m.Delta, 10))
	default:
		http.Error(w, "unknown metric type", http.StatusNotImplemented)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	val, getErr := h.Repo.Get(m.MType, m.ID)
	if getErr != nil {
		http.Error(w, getErr.Error(), http.StatusNotFound)
		return
	}

	switch m.MType {
	case "gauge":
		v, err := strconv.ParseFloat(val, 64)
		if err == nil {
			m.Value = &v
		}
	case "counter":
		v, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			m.Delta = &v
		}
	}

	json.NewEncoder(w).Encode(m)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m models.Metrics
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &m)
	if err != nil || m.ID == "" || m.MType == "" {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	val, getErr := h.Repo.Get(m.MType, m.ID)
	if getErr != nil {
		http.Error(w, getErr.Error(), http.StatusNotFound)
		return
	}

	switch m.MType {
	case "gauge":
		v, err := strconv.ParseFloat(val, 64)
		if err == nil {
			m.Value = &v
		}
	case "counter":
		v, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			m.Delta = &v
		}
	default:
		http.Error(w, "unknown metric type", http.StatusNotImplemented)
		return
	}

	json.NewEncoder(w).Encode(m)
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "metric name required", http.StatusBadRequest)
		return
	}

	if err := h.Repo.Add(metricType, name, value); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	value, err := h.Repo.Get(metricType, name)
	if err != nil {
		http.Error(w, "metric not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, value)
}

func (h *Handler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	metrics := h.Repo.GetAll()

	w.Write([]byte("<html><body><h1>Metrics</h1><ul>"))
	keys := make([]string, 0, len(metrics))
	for k := range metrics {
		keys = append(keys, k)
	}
	for _, k := range keys {
		v := metrics[k]
		w.Write([]byte("<li><b>" + k + "</b>: " + v + "</li>"))
	}
	w.Write([]byte("</ul></body></html>"))
}
