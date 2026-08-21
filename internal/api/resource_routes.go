package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sudo-su-coffee/firecracker-studio/internal/images"
	"github.com/sudo-su-coffee/firecracker-studio/internal/resources"
	"github.com/sudo-su-coffee/firecracker-studio/internal/worker"
	"github.com/zerodha/fastglue"
)

func resourceID(r *fastglue.Request) string { return fmt.Sprint(r.RequestCtx.UserValue("id")) }
func (s *Server) saveResourcesLocked() error {
	if s.resourceStore == nil {
		return fmt.Errorf("resource state store is not configured")
	}
	return s.resourceStore.Save([]resources.State{{MachineConfigs: s.machineConfigs, Kernels: s.kernels, Volumes: s.volumes, Vsocks: s.vsocks}})
}
func (s *Server) getMachineConfig(r *fastglue.Request) error {
	id := resourceID(r)
	s.resourceMu.RLock()
	c, ok := s.machineConfigs[id]
	s.resourceMu.RUnlock()
	if !ok {
		c = resources.MachineConfig{VCPUs: 1, MemoryMiB: 512, SMT: false}
	}
	return r.SendJSON(http.StatusOK, c)
}
func (s *Server) putMachineConfig(service *worker.Service) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var c resources.MachineConfig
		if err := r.Decode(&c, "json"); err != nil {
			return r.SendJSON(400, map[string]string{"error": err.Error()})
		}
		if err := resources.ValidateMachine(c); err != nil {
			return r.SendJSON(400, map[string]string{"error": err.Error()})
		}
		id := resourceID(r)
		c.UpdatedAt = time.Now().UTC()
		s.resourceMu.Lock()
		s.machineConfigs[id] = c
		err := s.saveResourcesLocked()
		s.resourceMu.Unlock()
		if err != nil {
			return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), s.workerTimeout)
		defer cancel()
		if err := service.ApplyMachineConfig(ctx, id, c.VCPUs, c.MemoryMiB, c.SMT); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "live_machine_config_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusOK, c)
	}
}
func (s *Server) patchMachineConfig(service *worker.Service) fastglue.FastRequestHandler {
	return s.putMachineConfig(service)
}
func (s *Server) constraints(r *fastglue.Request) error {
	return r.SendJSON(http.StatusOK, map[string]any{"maxVCPUs": 256, "minMemoryMiB": 128, "kvm": s.runtimeStatus.KVM})
}
func (s *Server) listKernels(r *fastglue.Request) error {
	s.resourceMu.RLock()
	defer s.resourceMu.RUnlock()
	out := make([]resources.Kernel, 0, len(s.kernels))
	for _, k := range s.kernels {
		out = append(out, k)
	}
	return r.SendJSON(http.StatusOK, map[string]any{"kernels": out})
}
func (s *Server) registerKernel(r *fastglue.Request) error {
	var k resources.Kernel
	if err := r.Decode(&k, "json"); err != nil {
		return r.SendJSON(400, map[string]string{"error": err.Error()})
	}
	if strings.TrimSpace(k.Path) == "" {
		return r.SendJSON(400, map[string]string{"error": "kernel path is required"})
	}
	if k.ID == "" {
		k.ID = fmt.Sprintf("kernel-%d", time.Now().UnixNano())
	}
	k.RegisteredAt = time.Now().UTC()
	s.resourceMu.Lock()
	s.kernels[k.ID] = k
	err := s.saveResourcesLocked()
	s.resourceMu.Unlock()
	if err != nil {
		return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return r.SendJSON(http.StatusCreated, k)
}
func (s *Server) deleteKernel(r *fastglue.Request) error {
	id := resourceID(r)
	s.resourceMu.Lock()
	defer s.resourceMu.Unlock()
	if _, ok := s.kernels[id]; !ok {
		return r.SendJSON(404, map[string]string{"error": "kernel_not_found"})
	}
	delete(s.kernels, id)
	if err := s.saveResourcesLocked(); err != nil {
		return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return r.SendJSON(http.StatusOK, map[string]string{"status": "deleted", "id": id})
}
func (s *Server) cloneImage(c *images.Catalog) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		id := resourceID(r)
		img, ok := c.Get(id)
		if !ok {
			return r.SendJSON(404, map[string]string{"error": "image_not_found"})
		}
		img.ID = fmt.Sprintf("%s-clone-%d", img.ID, time.Now().UnixNano())
		img.Digest = img.ID
		img.Reference = img.Reference + ":clone"
		img.CreatedAt = time.Time{}
		out, err := c.Upsert(img)
		if err != nil {
			return r.SendJSON(400, map[string]string{"error": err.Error()})
		}
		return r.SendJSON(http.StatusCreated, out)
	}
}
func (s *Server) pruneImages(c *images.Catalog) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var req struct {
			OlderThanHours  int  `json:"olderThanHours"`
			RemoveArtifacts bool `json:"removeArtifacts"`
		}
		if err := r.Decode(&req, "json"); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		}
		if req.OlderThanHours < 1 || req.OlderThanHours > 8760 {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "olderThanHours must be between 1 and 8760"})
		}
		removed, err := c.PruneFailedBefore(time.Now().UTC().Add(-time.Duration(req.OlderThanHours)*time.Hour), req.RemoveArtifacts)
		if err != nil {
			return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": "prune_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusOK, map[string]any{"status": "pruned", "removed": removed, "failedOnly": true, "removeArtifacts": req.RemoveArtifacts})
	}
}
func (s *Server) listVolumes(r *fastglue.Request) error {
	s.resourceMu.RLock()
	defer s.resourceMu.RUnlock()
	out := make([]resources.Volume, 0, len(s.volumes))
	for _, v := range s.volumes {
		out = append(out, v)
	}
	return r.SendJSON(http.StatusOK, map[string]any{"volumes": out})
}
func (s *Server) createVolume(r *fastglue.Request) error {
	var v resources.Volume
	if err := r.Decode(&v, "json"); err != nil {
		return r.SendJSON(400, map[string]string{"error": err.Error()})
	}
	if v.ID == "" {
		v.ID = fmt.Sprintf("vol-%d", time.Now().UnixNano())
	}
	v.Persistent = true
	v.CreatedAt = time.Now().UTC()
	s.resourceMu.Lock()
	s.volumes[v.ID] = v
	err := s.saveResourcesLocked()
	s.resourceMu.Unlock()
	if err != nil {
		return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return r.SendJSON(http.StatusCreated, v)
}
func (s *Server) deleteVolume(r *fastglue.Request) error {
	id := resourceID(r)
	s.resourceMu.Lock()
	defer s.resourceMu.Unlock()
	v, ok := s.volumes[id]
	if !ok {
		return r.SendJSON(404, map[string]string{"error": "volume_not_found"})
	}
	if v.AttachedVM != "" {
		return r.SendJSON(409, map[string]string{"error": "volume_attached"})
	}
	delete(s.volumes, id)
	if err := s.saveResourcesLocked(); err != nil {
		return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return r.SendJSON(http.StatusOK, map[string]string{"status": "deleted", "id": id})
}
func (s *Server) getVsock(r *fastglue.Request) error {
	id := resourceID(r)
	s.resourceMu.RLock()
	v := s.vsocks[id]
	s.resourceMu.RUnlock()
	return r.SendJSON(http.StatusOK, v)
}
func (s *Server) putVsock(service *worker.Service) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var v resources.VsockConfig
		if err := r.Decode(&v, "json"); err != nil {
			return r.SendJSON(400, map[string]string{"error": err.Error()})
		}
		if v.GuestCID == 0 {
			v.GuestCID = 3
		}
		id := resourceID(r)
		s.resourceMu.Lock()
		s.vsocks[id] = v
		err := s.saveResourcesLocked()
		s.resourceMu.Unlock()
		if err != nil {
			return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), s.workerTimeout)
		defer cancel()
		if err := service.ApplyVsock(ctx, id, v.GuestCID, v.HostPath); err != nil {
			return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "live_vsock_config_failed", "message": err.Error()})
		}
		return r.SendJSON(http.StatusOK, v)
	}
}
