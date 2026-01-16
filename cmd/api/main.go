package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/pressly/goose/v3"

	"urlShort/internal/config"
	httpapi "urlShort/internal/http"
	"urlShort/internal/http/handler"
	"urlShort/internal/repository/postgres"
	"urlShort/internal/service"
)

func main() {
	cfg := config.MustLoad()

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	if err := migrate(db, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	repo := postgres.NewLinksRepo(db)

	svc := service.NewLinksService(repo, service.Config{
		CodeLen:            7,
		ClickQueueSize:     cfg.ClickQueueSize,
		ClickWorkers:       cfg.ClickWorkers,
		ClickBatchSize:     cfg.ClickBatchSize,
		ClickFlushInterval: cfg.ClickFlushInterval,
		ClickWriteTimeout:  cfg.ClickWriteTimeout,
	}, log.Default())

	h := handler.NewLinksHandler(svc, cfg.BaseURL)
	router := httpapi.NewRouter(h)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr(),
		Handler:      router,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// 1) останавливаем HTTP (дожидаемся окончания активных handlers)
	_ = srv.Shutdown(shutdownCtx)

	// 2) теперь безопасно закрываем сервис (очередь кликов + воркеры)
	_ = svc.Shutdown(shutdownCtx)

	log.Printf("shutdown complete")
}

func migrate(db *sql.DB, dir string) error {
	goose.SetDialect("postgres")
	return goose.Up(db, dir)
}
