package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sudo-su-coffee/firecracker-studio/internal/api"
	"github.com/sudo-su-coffee/firecracker-studio/internal/config"
	"github.com/sudo-su-coffee/firecracker-studio/internal/converter"
	"github.com/sudo-su-coffee/firecracker-studio/internal/images"
	"github.com/sudo-su-coffee/firecracker-studio/internal/notifications"
	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
	"github.com/sudo-su-coffee/firecracker-studio/internal/runtime"
	"github.com/sudo-su-coffee/firecracker-studio/internal/web"
	"github.com/sudo-su-coffee/firecracker-studio/internal/worker"
	"github.com/valyala/fasthttp"
)

var version = "2.0.1"

func main() {
	cfg := config.Default()
	configPath := os.Getenv("FIRECRACKER_STUDIO_CONFIG")
	if configPath == "" {
		candidate := filepath.Join(cfg.StateDir, "config.toml")
		if _, err := os.Stat(candidate); err == nil {
			configPath = candidate
		}
	}
	if configPath != "" {
		loaded, err := config.Load(configPath)
		if err != nil {
			panic(err)
		}
		cfg = loaded
	} else {
		if address := os.Getenv("FIRECRACKER_STUDIO_LISTEN"); address != "" {
			cfg.ListenAddress = address
		}
		if err := cfg.Validate(); err != nil {
			panic(err)
		}
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log = log.With("service", cfg.AppName)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtimeManager := runtime.NewConfiguredManager(cfg.RuntimeRoot, time.Duration(cfg.RuntimeDownloadTimeout))
	runtimeStatus := runtimeManager.Status(ctx)
	defaultKernel := filepath.Join(runtimeManager.Root(), "images", "default", "vmlinux")
	imageConverter := converter.Hybrid{OCI: converter.OCI{ArtifactDir: cfg.ArtifactDir, Profile: converter.Profile{Name: "alpine", KernelPath: defaultKernel}}}
	ops, err := operations.NewManagerWithOptions(ctx, cfg.OperationWorkers, imageConverter, log, time.Duration(cfg.OperationTimeout), time.Duration(cfg.OperationRetention))
	if err != nil {
		log.Error("failed to initialize operation manager", "error", err)
		os.Exit(1)
	}
	catalog, err := images.NewCatalog(filepath.Join(cfg.StateDir, "images.json"))
	if err != nil {
		log.Error("failed to initialize image catalog", "error", err)
		os.Exit(1)
	}
	workerService, err := worker.NewConfiguredPersistentService(worker.DirectorySocketFactory{Dir: cfg.ArtifactDir, FirecrackerPath: runtimeStatus.Firecracker}, filepath.Join(cfg.ArtifactDir, "studio-vms.json"), time.Duration(cfg.FirecrackerAPITimeout), cfg.NetworkCIDR, cfg.GuestAgentPort, cfg.GuestAgentCID)
	if err != nil {
		log.Error("failed to initialize worker service", "error", err)
		os.Exit(1)
	}
	smtpPassword := cfg.Notifications.SMTPPassword
	if cfg.Notifications.Enabled && smtpPassword == "" && cfg.Notifications.SMTPPasswordFile != "" {
		password, readErr := os.ReadFile(cfg.Notifications.SMTPPasswordFile)
		if readErr != nil {
			log.Error("failed to read SMTP password file", "path", cfg.Notifications.SMTPPasswordFile, "error", readErr)
			os.Exit(1)
		}
		smtpPassword = strings.TrimSpace(string(password))
	}
	notifier := notifications.New(notifications.Config{Enabled: cfg.Notifications.Enabled, Host: cfg.Notifications.SMTPHost, Port: cfg.Notifications.SMTPPort, Username: cfg.Notifications.SMTPUsername, Password: smtpPassword, From: cfg.Notifications.From, Recipients: cfg.Notifications.Recipients})
	ops.SetFailureHook(func(op operations.Operation, failure error) {
		if err := notifier.Send("Firecracker Studio: image conversion failed", fmt.Sprintf("Operation %s for %s failed: %v", op.ID, op.Request.Source, failure)); err != nil {
			log.Error("conversion failure notification failed", "operation_id", op.ID, "error", err)
		}
	})
	apiServer, err := api.New(ops, catalog, workerService, runtimeStatus, defaultKernel, cfg, log, notifier)
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
