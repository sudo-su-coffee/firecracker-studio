package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sudo-su-coffee/firecracker-studio/internal/firecracker"
)

type SocketFactory interface {
	NewSocket(vmID string) (string, error)
}

type processLauncher interface {
	Launch(ctx context.Context, vmID, socket string) (*exec.Cmd, error)
}

type Service struct {
	factory   SocketFactory
	launcher  processLauncher
	network   *NetworkManager
	mu        sync.RWMutex
	vms       map[string]VM
	clients   map[string]*firecracker.Client
	processes map[string]*exec.Cmd
	nets      map[string]NetworkConfig
}

func NewService(factory SocketFactory) (*Service, error) {
	if factory == nil {
		return nil, fmt.Errorf("socket factory is required")
	}
	launcher, _ := factory.(processLauncher)
	return &Service{
		factory:   factory,
		launcher:  launcher,
		network:   NewNetworkManager(),
		vms:       make(map[string]VM),
		clients:   make(map[string]*firecracker.Client),
		processes: make(map[string]*exec.Cmd),
		nets:      make(map[string]NetworkConfig),
	}, nil
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
	if req.VCPUs > 32 {
		return VM{}, fmt.Errorf("vcpus must be between 1 and 32")
	}
	if req.MemoryMiB > 262144 {
		return VM{}, fmt.Errorf("memoryMiB must not exceed 262144")
	}
	for _, port := range req.PortMappings {
		if port.HostPort < 1 || port.HostPort > 65535 || port.GuestPort < 1 || port.GuestPort > 65535 {
			return VM{}, fmt.Errorf("port mappings must be between 1 and 65535")
		}
		if port.Protocol != "" && port.Protocol != "tcp" && port.Protocol != "udp" {
			return VM{}, fmt.Errorf("unsupported port protocol %q", port.Protocol)
		}
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

	vm := VM{ID: id, State: "created", ArtifactDigest: req.ArtifactDigest, ImageReference: req.ImageReference, KernelPath: req.KernelPath, RootfsPath: req.RootfsPath, VCPUs: req.VCPUs, MemoryMiB: req.MemoryMiB, PortMappings: append([]PortMapping(nil), req.PortMappings...), Logs: []string{"created workload", fmt.Sprintf("image %s", displayImage(req.ImageReference, req.ArtifactDigest))}, CreatedAt: now, UpdatedAt: now}
	if req.KernelPath != "" || req.RootfsPath != "" {
		if req.KernelPath == "" || req.RootfsPath == "" {
			return VM{}, fmt.Errorf("kernelPath and rootfsPath must be provided together")
		}
		if _, err := os.Stat(req.KernelPath); err != nil {
			return VM{}, fmt.Errorf("kernel path: %w", err)
		}
		if _, err := os.Stat(req.RootfsPath); err != nil {
			return VM{}, fmt.Errorf("rootfs path: %w", err)
		}
		if s.launcher == nil {
			return VM{}, fmt.Errorf("Firecracker process launcher is not configured")
		}
		proc, err := s.launcher.Launch(ctx, id, socket)
		if err != nil {
			return VM{}, fmt.Errorf("launch Firecracker: %w", err)
		}
		if err := waitForSocket(ctx, socket); err != nil {
			_ = proc.Process.Kill()
			return VM{}, err
		}
		if err := client.SetMachineConfig(ctx, firecracker.MachineConfig{VCPUCount: req.VCPUs, MemSizeMiB: req.MemoryMiB}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure machine: %w", err)
		}
		bootArgs := req.BootArgs
		if bootArgs == "" {
			bootArgs = "console=ttyS0 reboot=k panic=1 pci=off"
		}
		if err := client.SetBootSource(ctx, firecracker.BootSource{KernelImagePath: req.KernelPath, BootArgs: bootArgs}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure boot source: %w", err)
		}
		if err := client.SetDrive(ctx, firecracker.Drive{DriveID: "rootfs", PathOnHost: req.RootfsPath, IsRootDevice: true, IsReadOnly: false}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure rootfs: %w", err)
		}
		var netCfg NetworkConfig
		if len(req.PortMappings) > 0 || req.EnableNetworking {
			netCfg, err = s.network.Setup(ctx, id)
			if err != nil {
				_ = proc.Process.Kill()
				return VM{}, fmt.Errorf("configure networking: %w", err)
			}
			if err := client.SetNetworkInterface(ctx, firecracker.NetworkInterface{
				IfaceID:     netCfg.IfaceID,
				HostDevName: netCfg.TapDevice,
				GuestMac:    netCfg.GuestMac,
			}); err != nil {
				_ = s.network.Teardown(context.WithoutCancel(ctx), id)
				_ = proc.Process.Kill()
				return VM{}, fmt.Errorf("attach network interface: %w", err)
			}
			if err := client.SetBootSource(ctx, firecracker.BootSource{KernelImagePath: req.KernelPath, BootArgs: networkBootArgs(bootArgs, netCfg)}); err != nil {
				_ = s.network.Teardown(context.WithoutCancel(ctx), id)
				_ = proc.Process.Kill()
				return VM{}, fmt.Errorf("configure network boot arguments: %w", err)
			}
			vm.Network = &VMNetwork{TapDevice: netCfg.TapDevice, GuestMac: netCfg.GuestMac, GuestCIDR: netCfg.GuestCIDR, HostCIDR: netCfg.HostCIDR}

			s.mu.Lock()
			s.nets[id] = netCfg
			s.mu.Unlock()
			vm.Logs = append(vm.Logs, fmt.Sprintf("TAP device %s attached (guest %s)", netCfg.TapDevice, netCfg.GuestCIDR))
		}
		vm.Logs = append(vm.Logs, "Firecracker process launched", "kernel and rootfs configured")
		s.mu.Lock()
		s.processes[id] = proc
		s.mu.Unlock()
	}
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
	s.mu.RLock()
	proc := s.processes[id]
	netCfg, hasNet := s.nets[id]
	s.mu.RUnlock()
	if proc == nil && vm.KernelPath != "" {
		client, netCfg, err = s.relaunch(ctx, vm, hasNet)
		if err != nil {
			return s.appendLog(id, vm, "relaunch failed: "+err.Error()), err
		}
	}
	if err := client.Start(ctx); err != nil {
		return s.appendLog(id, vm, "start failed: "+err.Error()), err
	}
	vm = s.appendLog(id, s.update(id, "running", vm), "microVM started")
	if hasNet && len(vm.PortMappings) > 0 {
		guestIP := strings.SplitN(netCfg.GuestCIDR, "/", 2)[0]
		if err := s.network.ApplyPortMappings(ctx, guestIP, vm.PortMappings); err != nil {
			vm = s.appendLog(id, vm, "port mapping failed: "+err.Error())
			return vm, err
		}
		vm = s.appendLog(id, vm, fmt.Sprintf("port mappings applied (%d)", len(vm.PortMappings)))
	}
	return vm, nil
}

func (s *Service) relaunch(ctx context.Context, vm VM, enableNetworking bool) (*firecracker.Client, NetworkConfig, error) {
	if s.launcher == nil {
		return nil, NetworkConfig{}, fmt.Errorf("Firecracker process launcher is not configured")
	}
	socket, err := s.factory.NewSocket(vm.ID)
	if err != nil {
		return nil, NetworkConfig{}, fmt.Errorf("allocate VM socket: %w", err)
	}
	client, err := firecracker.NewClient(socket, 30*time.Second)
	if err != nil {
		return nil, NetworkConfig{}, err
	}
	proc, err := s.launcher.Launch(ctx, vm.ID, socket)
	if err != nil {
		return nil, NetworkConfig{}, fmt.Errorf("launch Firecracker: %w", err)
	}
	cleanup := func() { _ = proc.Process.Kill() }
	if err := waitForSocket(ctx, socket); err != nil {
		cleanup()
		return nil, NetworkConfig{}, err
	}
	vcpus, memory := vm.VCPUs, vm.MemoryMiB
	if vcpus <= 0 {
		vcpus = 1
	}
	if memory <= 0 {
		memory = 512
	}
	if err := client.SetMachineConfig(ctx, firecracker.MachineConfig{VCPUCount: vcpus, MemSizeMiB: memory}); err != nil {
		cleanup()
		return nil, NetworkConfig{}, err
	}
	bootArgs := "console=ttyS0 reboot=k panic=1 pci=off"
	if err := client.SetBootSource(ctx, firecracker.BootSource{KernelImagePath: vm.KernelPath, BootArgs: bootArgs}); err != nil {
		cleanup()
		return nil, NetworkConfig{}, err
	}
	if err := client.SetDrive(ctx, firecracker.Drive{DriveID: "rootfs", PathOnHost: vm.RootfsPath, IsRootDevice: true, IsReadOnly: false}); err != nil {
		cleanup()
		return nil, NetworkConfig{}, err
	}
	var netCfg NetworkConfig
	if enableNetworking {
		netCfg, err = s.network.Setup(ctx, vm.ID)
		if err != nil {
			cleanup()
			return nil, NetworkConfig{}, err
		}
		if err := client.SetNetworkInterface(ctx, firecracker.NetworkInterface{IfaceID: netCfg.IfaceID, HostDevName: netCfg.TapDevice, GuestMac: netCfg.GuestMac}); err != nil {
			_ = s.network.Teardown(context.WithoutCancel(ctx), vm.ID)
			cleanup()
			return nil, NetworkConfig{}, err
		}
		if err := client.SetBootSource(ctx, firecracker.BootSource{KernelImagePath: vm.KernelPath, BootArgs: networkBootArgs(bootArgs, netCfg)}); err != nil {
			_ = s.network.Teardown(context.WithoutCancel(ctx), vm.ID)
			cleanup()
			return nil, NetworkConfig{}, err
		}
		s.mu.Lock()
		s.nets[vm.ID] = netCfg
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.processes[vm.ID] = proc
	s.clients[vm.ID] = client
	s.mu.Unlock()
	return client, netCfg, nil
}

func (s *Service) Stop(ctx context.Context, id string) (VM, error) {
	client, vm, err := s.lookup(id)
	if err != nil {
		return VM{}, err
	}
	if err := client.SendCtrlAltDel(ctx); err != nil {
		return s.appendLog(id, vm, "stop failed: "+err.Error()), err
	}
	updated := s.appendLog(id, s.update(id, "stopping", vm), "shutdown requested")
	go func() {
		time.Sleep(2 * time.Second)
		s.mu.Lock()
		if proc := s.processes[id]; proc != nil && proc.Process != nil {
			_ = proc.Process.Kill()
			delete(s.processes, id)
		}
		netCfg, hasNet := s.nets[id]
		mappings := append([]PortMapping(nil), s.vms[id].PortMappings...)
		current := s.vms[id]
		current.State = "stopped"
		current.UpdatedAt = time.Now().UTC()
		s.vms[id] = current
		s.mu.Unlock()
		if hasNet {
			guestIP := strings.SplitN(netCfg.GuestCIDR, "/", 2)[0]
			s.network.RemovePortMappings(context.WithoutCancel(ctx), guestIP, mappings)
			_ = s.network.Teardown(context.WithoutCancel(ctx), id)
		}
	}()
	return updated, nil
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

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	vm, ok := s.vms[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("VM %q not found", id)
	}
	if proc := s.processes[id]; proc != nil && proc.Process != nil {
		_ = proc.Process.Kill()
	}
	netCfg, hasNet := s.nets[id]
	mappings := append([]PortMapping(nil), vm.PortMappings...)
	delete(s.processes, id)
	delete(s.clients, id)
	delete(s.vms, id)
	delete(s.nets, id)
	s.mu.Unlock()

	if hasNet {
		ctx := context.Background()
		guestIP := strings.SplitN(netCfg.GuestCIDR, "/", 2)[0]
		s.network.RemovePortMappings(ctx, guestIP, mappings)
		_ = s.network.Teardown(ctx, id)
	}
	if factory, ok := s.factory.(DirectorySocketFactory); ok {
		if err := os.RemoveAll(filepath.Join(factory.Dir, id)); err != nil {
			return fmt.Errorf("remove VM runtime directory: %w", err)
		}
	}
	return nil
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

func displayImage(reference, digest string) string {
	if reference != "" {
		return reference
	}
	return digest
}

func waitForSocket(ctx context.Context, socket string) error {
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		if _, err := os.Stat(socket); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("Firecracker socket did not appear: %s", socket)
		case <-time.After(25 * time.Millisecond):
		}
	}
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
	Dir             string
	FirecrackerPath string
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

func (f DirectorySocketFactory) Launch(ctx context.Context, vmID, socket string) (*exec.Cmd, error) {
	if f.FirecrackerPath == "" {
		return nil, fmt.Errorf("Firecracker binary path is not configured")
	}
	if _, err := os.Stat(f.FirecrackerPath); err != nil {
		return nil, fmt.Errorf("Firecracker binary: %w", err)
	}
	_ = os.Remove(socket)
	cmd := exec.CommandContext(context.WithoutCancel(ctx), f.FirecrackerPath, "--api-sock", socket)
	logPath := filepath.Join(filepath.Dir(socket), "firecracker.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open Firecracker log: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("start Firecracker: %w", err)
	}
	go func() { _ = cmd.Wait(); _ = logFile.Close() }()
	return cmd, nil
}
