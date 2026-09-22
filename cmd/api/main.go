package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"task-management/internal/adapters/database"
	"task-management/internal/adapters/redis"
	"task-management/internal/adapters/s3"
	"task-management/internal/config"
	"task-management/internal/controllers"
	"task-management/internal/repositories"
	"task-management/internal/routes"
	"task-management/internal/services"
)

func main() {
	cfg := config.Load()

	// Initialize database adapter
	db, err := database.NewPostgres(cfg)
	if err != nil {
		if cfg.AppEnv == "production" {
			log.Fatalf("failed to connect to postgres: %v", err)
		}
		log.Printf("[WARN] postgres connection failed: %v", err)
	} else {
		log.Println("postgres adapter connected successfully")
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("error closing postgres connection: %v", err)
			}
		}()
	}

	// Initialize redis adapter
	rdb, err := redis.New(cfg)
	if err != nil {
		if cfg.AppEnv == "production" {
			log.Fatalf("failed to connect to redis: %v", err)
		}
		log.Printf("[WARN] redis connection failed: %v", err)
	} else {
		log.Println("redis adapter connected successfully")
		defer func() {
			if err := rdb.Close(); err != nil {
				log.Printf("error closing redis connection: %v", err)
			}
		}()
	}

	// Initialize s3 storage adapter
	storage, err := s3.New(cfg)
	if err != nil {
		if cfg.AppEnv == "production" {
			log.Fatalf("failed to connect to s3: %v", err)
		}
		log.Printf("[WARN] s3 connection failed: %v", err)
	} else {
		log.Println("s3 adapter connected successfully")
	}

	repo := repositories.New(db.DB(), rdb)
	svc := services.New(cfg, repo, storage)
	ctrls := controllers.New(cfg, svc)
	router := routes.NewRouter(cfg, ctrls, svc)

	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("starting %s on port %s", cfg.AppName, cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}
