package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kosta324/metrics.git/internal/storage"
)

type Handler struct {
	Repo storage.Repository
}

func NewHandler(repo storage.Repository) *Handler {
	return &Handler{Repo: repo}
}

func (h *Handler) MainPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ожидается /update/{type}/{name}/{value}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 || parts[1] != "update" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	metricType := parts[2]
	name := parts[3]
	value := parts[4]

	if name == "" {
		http.Error(w, "metric name is required", http.StatusBadRequest)
		return
	}

	switch metricType {
	case "gauge":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
	case "counter":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported metric type", http.StatusBadRequest)
		return
	}

	if err := h.Repo.Add(metricType, name, value); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}
