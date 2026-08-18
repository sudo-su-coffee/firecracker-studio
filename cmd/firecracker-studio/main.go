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
	"github.com/sudo-su-coffee/firecracker-studio/internal/web"
	"github.com/sudo-su-coffee/firecracker-studio/internal/worker"
	"github.com/valyala/fasthttp"
)

var version = "dev"

func main() {
	cfg := config.Default()
	if address := os.Getenv("FIRECRACKER_STUDIO_LISTEN"); address != "" {
		cfg.ListenAddress = address
	}
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
	workerService, err := worker.NewService(worker.DirectorySocketFactory{Dir: cfg.ArtifactDir})
	if err != nil {
		log.Error("failed to initialize worker service", "error", err)
		os.Exit(1)
	}
	apiServer, err := api.New(ops, catalog, workerService, log)
	if err != nil {
		log.Error("failed to initialize api", "error", err)
		os.Exit(1)
	}

	log.Info("starting Firecracker Studio web server", "address", cfg.ListenAddress, "version", version)
	if err := fasthttp.ListenAndServe(cfg.ListenAddress, web.Handler(apiServer.Handler())); err != nil {
		log.Error("web server stopped", "error", err)
		os.Exit(1)
	}
}
