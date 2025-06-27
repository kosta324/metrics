package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kosta324/metrics.git/internal/handlers"
	"github.com/kosta324/metrics.git/internal/logger"
	"github.com/kosta324/metrics.git/internal/storage"
	"github.com/kosta324/metrics.git/internal/zipper"
	"go.uber.org/zap"
)

var (
	addr          = flag.String("a", "localhost:8080", "HTTP server address")
	storeInterval = flag.Int("i", 300, "Store interval in seconds (0 = sync write)")
	filePath      = flag.String("f", "/tmp/metrics-db.json", "File storage path")
	restore       = flag.Bool("r", true, "Restore metrics from file on startup")
)

var log zap.SugaredLogger

func parseEnvOverrides() {
	flag.Parse()
	if len(flag.Args()) > 0 {
		log.Fatalf("unknown arguments: %v", flag.Args())
	}
	if v := os.Getenv("ADDRESS"); v != "" {
		*addr = v
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			*storeInterval = i
		}
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		*filePath = v
	}
	if v := os.Getenv("RESTORE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			*restore = b
		}
	}
}

func main() {
	logging, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	log = *logging.Sugar()

	parseEnvOverrides()

	repo := storage.NewMemStorage()

	repo.SetFilePath(*filePath)

	if *restore {
		if err := repo.LoadFromFile(); err != nil {
			log.Warnf("failed to load metrics from file: %v", err)
		}
	}

	if *storeInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(*storeInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := repo.SaveToFile(); err != nil {
					log.Errorf("failed to save metrics: %v", err)
				}
			}
		}()
	}

	handler := handlers.NewHandler(repo)

	r := chi.NewRouter()

	r.Use(zipper.GzipMiddleware)
	r.Use(logger.WithLogging(&log))

	handler.RegisterRoutes(r)

	server := &http.Server{
		Addr:    *addr,
		Handler: r,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("Server running", zap.String("addr", *addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", zap.Error(err))
		}
	}()

	<-stop
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", zap.Error(err))
	}

	if err := repo.SaveToFile(); err != nil {
		log.Errorf("failed to save metrics on shutdown: %v", err)
	}
	log.Info("Server stopped gracefully")
}
