package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/kosta324/metrics.git/internal/agent"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	log := logger.Sugar()

	cfg := agent.Config{}
	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Poll interval (seconds)")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Report interval (seconds)")
	flag.Parse()

	cfg.ApplyEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		cancel()
	}()

	if err := agent.Run(ctx, cfg, log); err != nil {
		log.Fatalf("agent stopped with error: %v", err)
	}
}
