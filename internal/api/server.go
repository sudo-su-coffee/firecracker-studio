package api

import (
	"fmt"
	"github.com/sudo-su-coffee/firecracker-studio/internal/images"
	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"log/slog"
	"net/http"
)

type Server struct {
	app *fastglue.Fastglue
}

func New(ops *operations.Manager, catalog *images.Catalog, log *slog.Logger) (*Server, error) {
	if ops == nil {
		return nil, fmt.Errorf("operation manager is required")
	}
	if catalog == nil {
		return nil, fmt.Errorf("image catalog is required")
	}
	if log == nil {
		log = slog.Default()
	}
	app := fastglue.New()
	app.After(requestLogger(log))
	server := &Server{app: app}
	app.GET("/api/v1/health", server.health)
	app.GET("/api/v1/images", server.listImages(catalog))
	app.POST("/api/v1/images", server.registerImage(catalog))
	app.POST("/api/v1/conversions", server.enqueueConversion(ops))
	app.GET("/api/v1/operations", server.listOperations(ops))
	app.GET("/api/v1/operations/:id", server.getOperation(ops))
	app.NotFound(func(r *fastglue.Request) error {
		return r.SendJSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	})
	return server, nil
}

func (s *Server) Handler() func(*fasthttp.RequestCtx) { return s.app.Handler() }

func (s *Server) ListenAndServe(address string) error {
	return s.app.ListenAndServe(address, "", nil)
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
