package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sudo-su-coffee/firecracker-studio/internal/api"
	"github.com/sudo-su-coffee/firecracker-studio/internal/config"
	"github.com/sudo-su-coffee/firecracker-studio/internal/converter"
	"github.com/sudo-su-coffee/firecracker-studio/internal/images"
	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
)

func main() {
	cfg := config.Default()
	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log = log.With("service", cfg.AppName)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	imageConverter := converter.Hybrid{OCI: converter.OCI{ArtifactDir: cfg.ArtifactDir, Profile: converter.Profile{Name: "alpine"}}}
	ops, err := operations.NewManager(ctx, cfg.OperationWorkers, imageConverter, log)
	if err != nil {
		log.Error("failed to initialize operation manager", "error", err)
		os.Exit(1)
	}
	catalog := images.NewCatalog()
	server, err := api.New(ops, catalog, log)
	if err != nil {
		log.Error("failed to initialize api", "error", err)
		os.Exit(1)
	}
	log.Info("starting Firecracker Studio backend", "address", cfg.ListenAddress)
	if err := server.ListenAndServe(cfg.ListenAddress); err != nil {
		log.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
