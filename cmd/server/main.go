package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/handlers"
	"github.com/kosta324/metrics.git/internal/logger"
	"github.com/kosta324/metrics.git/internal/storage"
	"go.uber.org/zap"
)

var log zap.SugaredLogger

func main() {
	logging, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	log = *logging.Sugar()

	addr := flag.String("a", "localhost:8080", "HTTP server address")

	flag.Parse()
	if len(flag.Args()) > 0 {
		log.Fatalf("unknown arguments: %v", flag.Args())
	}

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}

	repo := storage.NewMemStorage()
	handler := handlers.NewHandler(repo)

	r := chi.NewRouter()

	r.Use(logger.WithLogging(&log))

	handler.RegisterRoutes(r)

	log.Info("Server running", zap.String("addr", *addr))
	err = http.ListenAndServe(*addr, r)
	if err != nil {
		log.Fatalf("Server failed: %v", zap.Error(err))
	}
}
