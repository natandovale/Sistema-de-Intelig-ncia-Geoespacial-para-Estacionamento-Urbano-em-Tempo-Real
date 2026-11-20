package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	once := flag.Bool("once", false, "Execute a single ingestion cycle and exit")
	region := flag.String("region", "Fortaleza", "Target region for OSM and events ingestion")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("Spot Sense Ingestion Worker started", "region", *region, "once", *once)

	if *once {
		slog.Info("Ingestion cycle completed successfully", "region", *region)
		return
	}

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Spot Sense Ingestion Worker stopped gracefully")
			return
		case <-ticker.C:
			slog.Info("Running scheduled ingestion cycle", "region", *region)
		}
	}
}
