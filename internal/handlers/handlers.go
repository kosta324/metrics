package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/storage"
)

type Handler struct {
	Repo storage.Repository
}

func NewHandler(repo storage.Repository) *Handler {
	return &Handler{Repo: repo}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Get("/", h.ListMetrics)
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
	metrics := h.Repo.GetAll()

	tmpl := `
		<html><body>
		<h1>Metrics</h1>
		<ul>
		{{ range $key, $val := . }}
			<li><b>{{ $key }}</b>: {{ $val }}</li>
		{{ end }}
		</ul>
		</body></html>
	`
	t, err := template.New("metrics").Parse(tmpl)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	t.Execute(w, metrics)
}
