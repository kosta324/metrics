package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/handlers"
	"github.com/kosta324/metrics.git/internal/storage"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")

	flag.Parse()
	if len(flag.Args()) > 0 {
		log.Fatalf("unknown arguments: %v", flag.Args())
	}

	repo := storage.NewMemStorage()
	handler := handlers.NewHandler(repo)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	log.Printf("Server running on %s", *addr)
	err := http.ListenAndServe(*addr, r)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
