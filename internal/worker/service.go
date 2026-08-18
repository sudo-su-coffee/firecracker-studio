package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sudo-su-coffee/firecracker-studio/internal/firecracker"
)

type SocketFactory interface {
	NewSocket(vmID string) (string, error)
}

type Service struct {
	factory SocketFactory
	mu      sync.RWMutex
	vms     map[string]VM
	clients map[string]*firecracker.Client
}

func NewService(factory SocketFactory) (*Service, error) {
	if factory == nil {
		return nil, fmt.Errorf("socket factory is required")
	}
	return &Service{factory: factory, vms: make(map[string]VM), clients: make(map[string]*firecracker.Client)}, nil
}

func (s *Service) Create(ctx context.Context, req VMRequest) (VM, error) {
	if req.ArtifactDigest == "" {
		return VM{}, fmt.Errorf("artifact digest is required")
	}
	if req.VCPUs <= 0 {
		req.VCPUs = 1
	}
	if req.MemoryMiB <= 0 {
		req.MemoryMiB = 512
	}
	id := uuid.NewString()
	socket, err := s.factory.NewSocket(id)
	if err != nil {
		return VM{}, fmt.Errorf("allocate VM socket: %w", err)
	}
	client, err := firecracker.NewClient(socket, 30*time.Second)
	if err != nil {
		return VM{}, err
	}
	now := time.Now().UTC()

	vm := VM{ID: id, State: "created", ArtifactDigest: req.ArtifactDigest, PortMappings: append([]PortMapping(nil), req.PortMappings...), Logs: []string{"created workload", fmt.Sprintf("artifact %s", req.ArtifactDigest)}, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.vms[id] = vm
	s.clients[id] = client
	s.mu.Unlock()
	return vm, nil
}

func (s *Service) Start(ctx context.Context, id string) (VM, error) {
	client, vm, err := s.lookup(id)
	if err != nil {
		return VM{}, err
	}
	if err := client.Start(ctx); err != nil {
		return s.appendLog(id, vm, "start failed: "+err.Error()), err
	}
	return s.appendLog(id, s.update(id, "running", vm), "microVM started"), nil
}

func (s *Service) Stop(ctx context.Context, id string) (VM, error) {
	client, vm, err := s.lookup(id)
	if err != nil {
		return VM{}, err
	}
	if err := client.SendCtrlAltDel(ctx); err != nil {
		return s.appendLog(id, vm, "stop failed: "+err.Error()), err
	}
	return s.appendLog(id, s.update(id, "stopping", vm), "shutdown requested"), nil
}

func (s *Service) CreateSnapshot(ctx context.Context, id, snapshotPath, memPath string) error {
	client, _, err := s.lookup(id)
	if err != nil {
		return err
	}
	if snapshotPath == "" || memPath == "" {
		return fmt.Errorf("snapshot and memory paths are required")
	}
	err = client.CreateSnapshot(ctx, firecracker.SnapshotCreate{SnapshotType: "Full", SnapshotPath: snapshotPath, MemFilePath: memPath})
	if vm, ok := s.Get(id); ok {
		message := "snapshot created"
		if err != nil {
			message = "snapshot failed: " + err.Error()
		}
		_ = s.appendLog(id, vm, message)
	}
	return err
}

func (s *Service) RestoreSnapshot(ctx context.Context, id, snapshotPath, memPath string) error {
	client, _, err := s.lookup(id)
	if err != nil {
		return err
	}
	if snapshotPath == "" || memPath == "" {
		return fmt.Errorf("snapshot and memory paths are required")
	}
	err = client.LoadSnapshot(ctx, firecracker.SnapshotLoad{SnapshotPath: snapshotPath, MemBackend: memPath})
	if vm, ok := s.Get(id); ok {
		message := "snapshot restored"
		if err != nil {
			message = "restore failed: " + err.Error()
		}
		_ = s.appendLog(id, vm, message)
	}
	return err
}

func (s *Service) Get(id string) (VM, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vm, ok := s.vms[id]
	return vm, ok
}

func (s *Service) List() []VM {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]VM, 0, len(s.vms))
	for _, vm := range s.vms {
		items = append(items, vm)
	}
	return items
}

func (s *Service) lookup(id string) (*firecracker.Client, VM, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[id]
	vm, exists := s.vms[id]
	if !ok || !exists {
		return nil, VM{}, fmt.Errorf("VM %q not found", id)
	}
	return client, vm, nil
}

func (s *Service) update(id, state string, vm VM) VM {
	vm.State = state
	vm.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	s.vms[id] = vm
	s.mu.Unlock()
	return vm
}

func (s *Service) appendLog(id string, vm VM, message string) VM {
	vm.Logs = append(vm.Logs, time.Now().UTC().Format(time.RFC3339)+" "+message)
	if len(vm.Logs) > 100 {
		vm.Logs = vm.Logs[len(vm.Logs)-100:]
	}
	vm.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	s.vms[id] = vm
	s.mu.Unlock()
	return vm
}

type DirectorySocketFactory struct {
	Dir string
}

func (f DirectorySocketFactory) NewSocket(vmID string) (string, error) {
	if f.Dir == "" {
		return "", fmt.Errorf("socket directory is required")
	}
	dir := filepath.Join(f.Dir, vmID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create VM directory: %w", err)
	}
	return filepath.Join(dir, "firecracker.sock"), nil
}
