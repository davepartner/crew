package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davepartner/crew/internal/config"
	"github.com/davepartner/crew/internal/httpapi"
	"github.com/davepartner/crew/internal/payauth"
	"github.com/davepartner/crew/internal/secret"
	"github.com/davepartner/crew/internal/store"
)

func main() {
	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.IsProduction() {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	client := payauth.NewClient(cfg.PaytrackifyURL, cfg.PaytrackifyTime)

	cipher, err := secret.LoadOrCreate(cfg.DBPath+".secret", cfg.SecretKey)
	if err != nil {
		slog.Error("load encryption key", "err", err)
		os.Exit(1)
	}

	server := httpapi.New(cfg, st, client, cipher)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		slog.Info("crew listening", "addr", cfg.Addr, "env", cfg.Env, "paytrackify", cfg.PaytrackifyURL)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("shutting down")
	_ = httpServer.Shutdown(ctx)
}
