package main

import (
	"context"
	"os/signal"
	"syscall"

	"musicclubbot/backend/internal/app"
	"musicclubbot/backend/internal/config"
	"musicclubbot/backend/internal/db"

	"os"

	"github.com/apsdehal/go-logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logger.SetDefaultFormat("%{time} %{lvl} %{file}:%{line} %{message}")
	log, _ := logger.New("", 1, os.Stdout)
	ctx = context.WithValue(ctx, "log", log)
	ctx = context.WithValue(ctx, "cfg", cfg)
	ctx = context.WithValue(ctx, "db", db.MustInitDb(ctx, cfg.DbUrl))

	if err := app.Run(ctx); err != nil {
		log.Fatalf("backend exited with error: %v", err)
	}
}
