package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"urlShort/internal/config"
	httpx "urlShort/internal/http"
	"urlShort/internal/repository/postgres"
)

func main() {
	cfg := config.MustLoad()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, dbErr := postgres.Connect(ctx, cfg.DatabaseUrl)
	if dbErr != nil {
		log.Fatalf("Database connection is failed: %v", dbErr)
	}
	defer db.Close()

	router := httpx.NewRouter()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server failed: %v", err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown err: %v", err)
	}
	log.Printf("bye bye")
}
