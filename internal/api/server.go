package api

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sudo-su-coffee/firecracker-studio/internal/images"
	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
	studiort "github.com/sudo-su-coffee/firecracker-studio/internal/runtime"
	"github.com/sudo-su-coffee/firecracker-studio/internal/worker"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"log/slog"
	"net/http"
)

type Server struct {
	app           *fastglue.Fastglue
	ops           *operations.Manager
	catalog       *images.Catalog
	workers       *worker.Service
	runtimeStatus studiort.Status
	eventsMu      sync.RWMutex
	events        []string
}

type hostMetrics struct {
	CPUPercent       float64  `json:"cpuPercent"`
	MemoryTotalBytes uint64   `json:"memoryTotalBytes"`
	MemoryUsedBytes  uint64   `json:"memoryUsedBytes"`
	MemoryPercent    float64  `json:"memoryPercent"`
	DiskTotalBytes   uint64   `json:"diskTotalBytes"`
	DiskUsedBytes    uint64   `json:"diskUsedBytes"`
	DiskPercent      float64  `json:"diskPercent"`
	NetworkRxBytes   uint64   `json:"networkRxBytes"`
	NetworkTxBytes   uint64   `json:"networkTxBytes"`
	GPUAvailable     bool     `json:"gpuAvailable"`
	GPUUsagePercent  *float64 `json:"gpuUsagePercent"`
	GPUMessage       string   `json:"gpuMessage"`
}

type metricsSnapshot struct {
	Timestamp  time.Time   `json:"timestamp"`
	Workers    int         `json:"workers"`
	MicroVMs   int         `json:"microvms"`
	RunningVMs int         `json:"runningVms"`
	Images     int         `json:"images"`
	Operations int         `json:"operations"`
	Healthy    bool        `json:"healthy"`
	Host       hostMetrics `json:"host"`
}

func New(ops *operations.Manager, catalog *images.Catalog, workers *worker.Service, runtimeStatus studiort.Status, log *slog.Logger) (*Server, error) {
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
	server := &Server{app: app, ops: ops, catalog: catalog, workers: workers, runtimeStatus: runtimeStatus, events: make([]string, 0, 100)}
	app.After(server.requestLogger(log))
	app.GET("/api/v1/health", server.health)
	app.GET("/api/v1/metrics", server.metrics)
	app.GET("/api/v1/logs", server.logs)
	app.GET("/api/v1/base-images", server.listBaseImages)
	app.GET("/api/v1/readiness", server.readiness)
	app.GET("/api/v1/images", server.listImages(catalog))
	app.POST("/api/v1/images", server.registerImage(catalog))
	app.POST("/api/v1/conversions", server.enqueueConversion(ops, catalog))
	app.GET("/api/v1/operations", server.listOperations(ops))
	app.GET("/api/v1/vms", server.listVMs(workers))
	app.POST("/api/v1/vms", server.createVM(workers))
	app.POST("/api/v1/vms/{id}/start", server.startVM(workers))
	app.POST("/api/v1/vms/{id}/stop", server.stopVM(workers))
	app.DELETE("/api/v1/vms/{id}", server.deleteVM(workers))
	app.POST("/api/v1/vms/{id}/snapshots", server.createSnapshot(workers))
	app.POST("/api/v1/vms/{id}/snapshots/restore", server.restoreSnapshot(workers))
	app.GET("/api/v1/operations/{id}", server.getOperation(ops))
	app.NotFound(func(r *fastglue.Request) error {
		return r.SendJSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	})
	return server, nil
}

func (s *Server) Handler() func(*fasthttp.RequestCtx) {
	handler := s.app.Handler()
	return func(ctx *fasthttp.RequestCtx) {
		if !s.authorized(ctx) {
			ctx.SetStatusCode(http.StatusUnauthorized)
			ctx.Response.Header.SetContentType("application/json")
			ctx.SetBodyString(`{"error":"unauthorized","message":"provide FIRECRACKER_STUDIO_TOKEN as a Bearer token"}`)
			return
		}
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

func (s *Server) authorized(ctx *fasthttp.RequestCtx) bool {
	token := strings.TrimSpace(os.Getenv("FIRECRACKER_STUDIO_TOKEN"))
	if token == "" {
		return true
	}
	if string(ctx.Path()) == "/api/v1/health" && ctx.IsGet() {
		return true
	}
	want := "Bearer " + token
	got := string(ctx.Request.Header.Peek("Authorization"))
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (s *Server) requestLogger(log *slog.Logger) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		method := string(r.RequestCtx.Method())
		path := string(r.RequestCtx.Path())
		message := time.Now().UTC().Format(time.RFC3339) + " " + method + " " + path
		log.Info("http request", "method", method, "path", path)
		s.eventsMu.Lock()
		s.events = append(s.events, message)
		if len(s.events) > 100 {
			s.events = s.events[len(s.events)-100:]
		}
		s.eventsMu.Unlock()
		return r
	}
}

func (s *Server) health(r *fastglue.Request) error {
	return r.SendJSON(http.StatusOK, map[string]string{"status": "ok", "service": "firecracker-studio", "version": "1.4.0"})
}

func (s *Server) readiness(r *fastglue.Request) error {
	status := s.runtimeStatus
	ready := status.Installed && status.KVM == "ready" && status.Kernel == "present" && status.Rootfs == "present"
	return r.SendJSON(http.StatusOK, map[string]any{"ready": ready, "runtime": status, "message": status.Message})
}

func (s *Server) logs(r *fastglue.Request) error {
	s.eventsMu.RLock()
	items := append([]string(nil), s.events...)
	s.eventsMu.RUnlock()
	return r.SendJSON(http.StatusOK, map[string]any{"events": items})
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
		Images: len(s.catalog.List()), Operations: len(s.ops.List()), Healthy: true, Host: readHostMetrics(),
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

func (s *Server) enqueueConversion(ops *operations.Manager, catalog *images.Catalog) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var req operations.Request
		if err := r.Decode(&req, "json"); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		}
		op, err := ops.Enqueue(r.RequestCtx, req)
		if err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "enqueue_failed", "message": err.Error()})
		}
		go s.registerCompletedImage(op.ID, ops, catalog)
		return r.SendJSON(http.StatusAccepted, op)
	}
}

func (s *Server) registerCompletedImage(id string, ops *operations.Manager, catalog *images.Catalog) {
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		op, ok := ops.Get(id)
		if !ok {
			return
		}
		if op.State == operations.StateSucceeded && op.Artifact != nil {
			_, _ = catalog.Upsert(images.Image{ID: op.Artifact.Digest, Reference: op.Request.Source, SourceType: op.Request.SourceType, Digest: op.Artifact.Digest, Architecture: op.Request.Architecture, BaseProfile: op.Request.BaseProfile, ArtifactPath: op.Artifact.Rootfs, Verified: true})
			return
		}
		if op.State == operations.StateFailed {
			return
		}
		time.Sleep(500 * time.Millisecond)
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

func (s *Server) deleteVM(service *worker.Service) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		id := fmt.Sprint(r.RequestCtx.UserValue("id"))
		if id == "" || id == "<nil>" {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "vm_id_required"})
		}
		if err := service.Delete(id); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "vm_delete_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusOK, map[string]string{"status": "deleted", "vmId": id})
	}
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
