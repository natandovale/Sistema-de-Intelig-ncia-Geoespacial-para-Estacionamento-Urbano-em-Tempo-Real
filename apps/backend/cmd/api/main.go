package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deliveryhttp "github.com/spot-sense/backend/internal/delivery/http"
	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/repository/postgres"
	redisrepo "github.com/spot-sense/backend/internal/repository/redis"
	"github.com/spot-sense/backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var cache domain.SpotCache
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		slog.Info("initializing Redis cache", "redis_url", redisURL)
		cache = redisrepo.NewRedisSpotCache(redisURL, 10*time.Minute)
	} else {
		slog.Info("initializing in-memory grid cache (10m TTL)")
		cache = redisrepo.NewMemorySpotCache(10 * time.Minute)
	}

	spatialRepo := postgres.NewSeedSpatialRepository()
	searchService := service.NewSearchSpotsService(spatialRepo, cache)
	router := deliveryhttp.NewRouter(searchService)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Spot Sense API listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	slog.Info("Spot Sense API stopped gracefully")
}
