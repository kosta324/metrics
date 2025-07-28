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
	_ "github.com/jackc/pgx/v5/stdlib"
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
	dbDSN         = flag.String("d", "", "PostgreSQL DSN")
)

var log zap.SugaredLogger

func parseEnvOverrides() {
	flag.Parse()
	if len(flag.Args()) > 0 {
		log.Fatalf("unknown arguments: %v", flag.Args())
	}
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		*addr = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if i, err := strconv.Atoi(v); err == nil {
			*storeInterval = i
		}
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		*filePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			*restore = b
		}
	}
	if envDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		*dbDSN = envDSN
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

	var repo storage.Repository
	var sqlDB *storage.SQLRepo
	if *dbDSN != "" {
		sqlDB, err = storage.NewSQLStorage(*dbDSN)
		if err != nil {
			log.Fatalf("failed to connect to DB: %v", err)
		}
		repo = sqlDB
	} else if *filePath != "" {
		memRepo := storage.NewMemStorage()
		memRepo.SetFilePath(*filePath)
		if *restore {
			if err := memRepo.LoadFromFile(); err != nil {
				log.Warnf("failed to load metrics: %v", err)
			}
		}
		repo = memRepo
		if *storeInterval > 0 {
			go func() {
				ticker := time.NewTicker(time.Duration(*storeInterval) * time.Second)
				defer ticker.Stop()
				for range ticker.C {
					if err := memRepo.SaveToFile(); err != nil {
						log.Errorf("failed to save metrics: %v", err)
					}
				}
			}()
		}
	} else {
		repo = storage.NewMemStorage()
	}

	r := chi.NewRouter()

	r.Use(zipper.GzipMiddleware)
	r.Use(logger.WithLogging(&log))

	handler := handlers.NewHandler(repo, &log)
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

	if memRepo, ok := repo.(*storage.MemStorage); ok && *filePath != "" {
		if err := memRepo.SaveToFile(); err != nil {
			log.Errorf("failed to save metrics on shutdown: %v", err)
		}
	}

	log.Info("Server stopped gracefully")
}
