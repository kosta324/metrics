package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/handlers"
	"github.com/kosta324/metrics.git/internal/storage"
)

func main() {
	repo := storage.NewMemStorage()
	handler := handlers.NewHandler(repo)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	log.Println("Server running on :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
