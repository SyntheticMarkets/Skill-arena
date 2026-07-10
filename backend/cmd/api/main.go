package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/server"
	"skill-arena/internal/workers"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Printf("payment provider status: %+v", cfg.Settings.Payments.ProviderStatus())

	store, err := db.NewWithOptions(ctx, db.Options{
		DatabaseURL: cfg.DatabaseURL,
		Environment: cfg.Environment,
		RedisURL:    cfg.RedisURL,
		Storage:     cfg.Settings.Storage,
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	handler := server.New(store, cfg)
	addr := cfg.HTTPAddr
	if addr == "" {
		addr = ":8080"
	}
	handler.Addr = addr

	workerManager := workers.NewManager(store, cfg)
	workerManager.Start(ctx)

	serverErrors := make(chan error, 1)
	log.Printf("starting Skill Arena API on %s", addr)
	go func() {
		serverErrors <- handler.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}

	shutdownSeconds := cfg.Settings.Workers.ShutdownSeconds
	if shutdownSeconds <= 0 {
		shutdownSeconds = 20
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownSeconds)*time.Second)
	defer cancel()

	log.Printf("stopping workers")
	if err := workerManager.Shutdown(shutdownCtx); err != nil {
		log.Printf("worker shutdown warning: %v", err)
	}

	log.Printf("stopping HTTP server")
	if err := handler.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown warning: %v", err)
	}

	if err := store.Close(shutdownCtx); err != nil {
		log.Printf("store close warning: %v", err)
	}
}
