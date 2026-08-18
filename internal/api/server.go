package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sudo-su-coffee/firecracker-studio/internal/images"
	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
	"github.com/sudo-su-coffee/firecracker-studio/internal/worker"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"log/slog"
	"net/http"
)

type Server struct {
	app         *fastglue.Fastglue
	ops         *operations.Manager
	catalog     *images.Catalog
	workers     *worker.Service
	marketplace *images.Marketplace
}

type metricsSnapshot struct {
	Timestamp  time.Time `json:"timestamp"`
	Workers    int       `json:"workers"`
	MicroVMs   int       `json:"microvms"`
	RunningVMs int       `json:"runningVms"`
	Images     int       `json:"images"`
	Operations int       `json:"operations"`
	Healthy    bool      `json:"healthy"`
}

func New(ops *operations.Manager, catalog *images.Catalog, workers *worker.Service, log *slog.Logger) (*Server, error) {
	if ops == nil {
		return nil, fmt.Errorf("operation manager is required")
	}
	if catalog == nil {
		return nil, fmt.Errorf("image catalog is required")
	}
	if workers == nil {
		return nil, fmt.Errorf("worker service is required")
	}
	if log == nil {
		log = slog.Default()
	}
	app := fastglue.New()
	app.After(requestLogger(log))
	base, _ := os.UserConfigDir()
	runtimeRoot := filepath.Join(base, "FirecrackerStudio", "runtime")
	server := &Server{app: app, ops: ops, catalog: catalog, workers: workers, marketplace: images.NewMarketplace(runtimeRoot, os.Getenv("FIRECRACKER_STUDIO_MARKETPLACE_URL"))}
	app.GET("/api/v1/health", server.health)
	app.GET("/api/v1/metrics", server.metrics)
	app.GET("/api/v1/base-images", server.listBaseImages)
	app.GET("/api/v1/marketplace/images", server.listMarketplaceImages)
	app.POST("/api/v1/marketplace/images/:id/pull", server.pullMarketplaceImage)
	app.GET("/api/v1/images", server.listImages(catalog))
	app.POST("/api/v1/images", server.registerImage(catalog))
	app.POST("/api/v1/conversions", server.enqueueConversion(ops))
	app.GET("/api/v1/operations", server.listOperations(ops))
	app.GET("/api/v1/vms", server.listVMs(workers))
	app.POST("/api/v1/vms", server.createVM(workers))
	app.POST("/api/v1/vms/:id/start", server.startVM(workers))
	app.POST("/api/v1/vms/:id/stop", server.stopVM(workers))
	app.POST("/api/v1/vms/:id/snapshots", server.createSnapshot(workers))
	app.POST("/api/v1/vms/:id/snapshots/restore", server.restoreSnapshot(workers))
	app.GET("/api/v1/operations/:id", server.getOperation(ops))
	app.NotFound(func(r *fastglue.Request) error {
		return r.SendJSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	})
	return server, nil
}

func (s *Server) Handler() func(*fasthttp.RequestCtx) {
	handler := s.app.Handler()
	return func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/api/v1/metrics/stream" {
			s.metricsStream(ctx)
			return
		}
		origin := string(ctx.Request.Header.Peek("Origin"))
		allowed := os.Getenv("FIRECRACKER_STUDIO_CORS_ORIGIN")
		if allowed == "" {
			allowed = "http://localhost:5173"
		}
		if origin == allowed {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			ctx.Response.Header.Set("Vary", "Origin")
		}
		if ctx.IsOptions() {
			ctx.SetStatusCode(http.StatusNoContent)
			return
		}
		handler(ctx)
	}
}

func (s *Server) ListenAndServe(address string) error {
	return fasthttp.ListenAndServe(address, s.Handler())
}

func requestLogger(log *slog.Logger) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		log.Info("http request", "method", string(r.RequestCtx.Method()), "path", string(r.RequestCtx.Path()))
		return r
	}
}

func (s *Server) health(r *fastglue.Request) error {
	return r.SendJSON(http.StatusOK, map[string]string{"status": "ok", "service": "firecracker-studio"})
}

func (s *Server) snapshot() metricsSnapshot {
	vms := s.workers.List()
	running := 0
	for _, vm := range vms {
		if vm.State == "running" {
			running++
		}
	}
	return metricsSnapshot{
		Timestamp: time.Now().UTC(), Workers: 1, MicroVMs: len(vms), RunningVMs: running,
		Images: len(s.catalog.List()), Operations: len(s.ops.List()), Healthy: true,
	}
}

func (s *Server) metrics(r *fastglue.Request) error {
	return r.SendJSON(http.StatusOK, s.snapshot())
}

func (s *Server) metricsStream(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			data, err := json.Marshal(s.snapshot())
			if err != nil {
				return
			}
			if _, err = w.WriteString("event: metrics\ndata: " + string(data) + "\n\n"); err != nil {
				return
			}
			if err = w.Flush(); err != nil {
				return
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	})
}

func (s *Server) listBaseImages(r *fastglue.Request) error {
	return r.SendJSON(http.StatusOK, map[string]any{"images": images.ManagedBaseImages()})
}

func (s *Server) listMarketplaceImages(r *fastglue.Request) error {
	items, err := s.marketplace.List(r.RequestCtx)
	if err != nil {
		return r.SendJSON(http.StatusBadGateway, map[string]string{"error": "marketplace_unavailable", "message": err.Error()})
	}
	return r.SendJSON(http.StatusOK, map[string]any{"images": items, "source": s.marketplace.CatalogURL})
}

func (s *Server) pullMarketplaceImage(r *fastglue.Request) error {
	id := fmt.Sprint(r.RequestCtx.UserValue("id"))
	if id == "" || id == "<nil>" {
		return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "image_id_required"})
	}
	item, dir, err := s.marketplace.Pull(r.RequestCtx, id)
	if err != nil {
		return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "image_pull_failed", "message": err.Error()})
	}
	digest := "sha256:" + item.KernelSHA256
	stored, err := s.catalog.Upsert(images.Image{ID: item.ID, Reference: item.Name, SourceType: "marketplace", Digest: digest, Architecture: item.Architecture, BaseProfile: item.Distribution, ArtifactPath: dir, Verified: true})
	if err != nil {
		return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": "image_register_failed", "message": err.Error()})
	}
	return r.SendJSON(http.StatusCreated, map[string]any{"image": stored, "marketplace": item, "directory": dir, "status": "downloaded"})
}

func (s *Server) listImages(catalog *images.Catalog) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		return r.SendJSON(http.StatusOK, map[string]any{"images": catalog.List()})
	}
}

func (s *Server) registerImage(catalog *images.Catalog) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var image images.Image
		if err := r.Decode(&image, "json"); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		}
		stored, err := catalog.Upsert(image)
		if err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_image", "message": err.Error()})
		}
		return r.SendJSON(http.StatusCreated, stored)
	}
}

func (s *Server) enqueueConversion(ops *operations.Manager) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var req operations.Request
		if err := r.Decode(&req, "json"); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		}
		op, err := ops.Enqueue(r.RequestCtx, req)
		if err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "enqueue_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusAccepted, op)
	}
}

func (s *Server) listVMs(service *worker.Service) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		return r.SendJSON(http.StatusOK, map[string]any{"vms": service.List()})
	}
}

func (s *Server) createVM(service *worker.Service) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var req worker.VMRequest
		if err := r.Decode(&req, "json"); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		}
		vm, err := service.Create(r.RequestCtx, req)
		if err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "create_vm_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusCreated, vm)
	}
}

func (s *Server) startVM(service *worker.Service) fastglue.FastRequestHandler {
	return s.vmAction(service, true)
}

func (s *Server) stopVM(service *worker.Service) fastglue.FastRequestHandler {
	return s.vmAction(service, false)
}

type snapshotRequest struct {
	SnapshotPath string `json:"snapshotPath"`
	MemoryPath   string `json:"memoryPath"`
}

func (s *Server) createSnapshot(service *worker.Service) fastglue.FastRequestHandler {
	return s.snapshotAction(service, false)
}

func (s *Server) restoreSnapshot(service *worker.Service) fastglue.FastRequestHandler {
	return s.snapshotAction(service, true)
}

func (s *Server) snapshotAction(service *worker.Service, restore bool) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		id := fmt.Sprint(r.RequestCtx.UserValue("id"))
		var req snapshotRequest
		if err := r.Decode(&req, "json"); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		}
		var err error
		if restore {
			err = service.RestoreSnapshot(r.RequestCtx, id, req.SnapshotPath, req.MemoryPath)
		} else {
			err = service.CreateSnapshot(r.RequestCtx, id, req.SnapshotPath, req.MemoryPath)
		}
		if err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "snapshot_operation_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusAccepted, map[string]string{"status": "accepted", "vmId": id})
	}
}

func (s *Server) vmAction(service *worker.Service, start bool) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		id := fmt.Sprint(r.RequestCtx.UserValue("id"))
		if id == "" || id == "<nil>" {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "vm_id_required"})
		}
		var vm worker.VM
		var err error
		if start {
			vm, err = service.Start(r.RequestCtx, id)
		} else {
			vm, err = service.Stop(r.RequestCtx, id)
		}
		if err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "vm_action_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusOK, vm)
	}
}

func (s *Server) listOperations(ops *operations.Manager) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		return r.SendJSON(http.StatusOK, map[string]any{"operations": ops.List()})
	}
}

func (s *Server) getOperation(ops *operations.Manager) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		id := fmt.Sprint(r.RequestCtx.UserValue("id"))
		if id == "" || id == "<nil>" {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "operation_id_required"})
		}
		op, ok := ops.Get(id)
		if !ok {
			return r.SendJSON(http.StatusNotFound, map[string]string{"error": "operation_not_found"})
		}
		return r.SendJSON(http.StatusOK, op)
	}
}
