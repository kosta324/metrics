package main

import (
	"log"
	"net/http"

	"github.com/kosta324/metrics.git/internal/handlers"
	"github.com/kosta324/metrics.git/internal/storage"
)

func main() {
	repo := storage.NewMemStorage()
	handler := handlers.NewHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/{type}/{name}/{val}`, handler.MainPage)

	log.Println("Server running on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
