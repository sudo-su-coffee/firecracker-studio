package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sudo-su-coffee/firecracker-studio/internal/firecracker"
	"github.com/sudo-su-coffee/firecracker-studio/internal/state"
)

type SocketFactory interface {
	NewSocket(vmID string) (string, error)
}
type processLauncher interface {
	Launch(ctx context.Context, vmID, socket string) (*exec.Cmd, error)
}

type Service struct {
	factory        SocketFactory
	launcher       processLauncher
	clientTimeout  time.Duration
	guestAgentPort uint32
	guestAgentCID  uint32
	mu             sync.RWMutex
	vms            map[string]VM
	clients        map[string]*firecracker.Client
	processes      map[string]*exec.Cmd
	store          *state.Store[VM]
	network        *NetworkManager
}

func NewService(factory SocketFactory) (*Service, error) {
	return newConfiguredService(factory, nil, 30*time.Second, "172.16.0.0/16", 5000, 3)
}

func NewPersistentService(factory SocketFactory, statePath string) (*Service, error) {
	return NewConfiguredPersistentService(factory, statePath, 30*time.Second, "172.16.0.0/16", 5000, 3)
}

func NewConfiguredPersistentService(factory SocketFactory, statePath string, clientTimeout time.Duration, networkCIDR string, guestAgentPort, guestAgentCID uint32) (*Service, error) {
	store, err := state.New[VM](statePath)
	if err != nil {
		return nil, err
	}
	return newConfiguredService(factory, store, clientTimeout, networkCIDR, guestAgentPort, guestAgentCID)
}

func newService(factory SocketFactory, store *state.Store[VM]) (*Service, error) {
	return newConfiguredService(factory, store, 30*time.Second, "172.16.0.0/16", 5000, 3)
}

func newConfiguredService(factory SocketFactory, store *state.Store[VM], clientTimeout time.Duration, networkCIDR string, guestAgentPort, guestAgentCID uint32) (*Service, error) {
	if factory == nil {
		return nil, fmt.Errorf("socket factory is required")
	}
	launcher, _ := factory.(processLauncher)
	if clientTimeout <= 0 {
		clientTimeout = 30 * time.Second
	}
	if guestAgentPort == 0 {
		guestAgentPort = 5000
	}
	if guestAgentCID == 0 {
		guestAgentCID = 3
	}
	network := NewNetworkManager()
	network.BridgeCIDRBase = networkCIDR
	s := &Service{factory: factory, launcher: launcher, clientTimeout: clientTimeout, guestAgentPort: guestAgentPort, guestAgentCID: guestAgentCID, vms: map[string]VM{}, clients: map[string]*firecracker.Client{}, processes: map[string]*exec.Cmd{}, store: store, network: network}
	if store != nil {
		items, err := store.Load()
		if err != nil {
			return nil, err
		}
		for _, vm := range items {
			vm.State = "unknown"
			vm.Logs = append(vm.Logs, time.Now().UTC().Format(time.RFC3339)+" recovered after Studio restart; process liveness must be checked")
			s.vms[vm.ID] = vm
			if vm.SocketPath != "" {
				if client, err := firecracker.NewClient(vm.SocketPath, s.clientTimeout); err == nil {
					s.clients[vm.ID] = client
				}
			}
		}
		_ = s.persist()
	}
	return s, nil
}

func (s *Service) ExecGuest(ctx context.Context, id, command string) ([]byte, error) {
	if command == "" {
		return nil, fmt.Errorf("guest command is required")
	}
	if len(command) > 4096 {
		return nil, fmt.Errorf("guest command is too long")
	}
	client, _, err := s.lookup(id)
	if err != nil {
		return nil, err
	}
	conn, err := client.OpenVsock(ctx, s.guestAgentPort)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := fmt.Fprintf(conn, "%s\n", command); err != nil {
		return nil, fmt.Errorf("send guest command: %w", err)
	}
	return io.ReadAll(io.LimitReader(conn, 1<<20))
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
	storageMode := req.StorageMode
	if storageMode == "" {
		storageMode = "ephemeral"
	}
	if storageMode != "ephemeral" && storageMode != "persistent" {
		return VM{}, fmt.Errorf("storageMode must be persistent or ephemeral")
	}
	id := uuid.NewString()
	socket, err := s.factory.NewSocket(id)
	if err != nil {
		return VM{}, fmt.Errorf("allocate VM socket: %w", err)
	}
	client, err := firecracker.NewClient(socket, s.clientTimeout)
	if err != nil {
		return VM{}, err
	}
	now := time.Now().UTC()
	vm := VM{ID: id, State: "created", ArtifactDigest: req.ArtifactDigest, ImageReference: req.ImageReference, KernelPath: req.KernelPath, RootfsPath: req.RootfsPath, SocketPath: socket, StorageMode: storageMode, PersistentDisk: req.PersistentDisk, PortMappings: append([]PortMapping(nil), req.PortMappings...), Logs: []string{"created workload", fmt.Sprintf("image %s", displayImage(req.ImageReference, req.ArtifactDigest))}, CreatedAt: now, UpdatedAt: now}
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
		networkConfig, networkErr := s.network.Setup(ctx, id)
		if networkErr != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure VM networking: %w", networkErr)
		}
		vm.TapDevice, vm.GuestIP, vm.HostIP, vm.GuestMAC = networkConfig.TapDevice, networkConfig.GuestCIDR, networkConfig.HostCIDR, networkConfig.GuestMac
		if err := client.SetMachineConfig(ctx, firecracker.MachineConfig{VCPUCount: req.VCPUs, MemSizeMiB: req.MemoryMiB}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure machine: %w", err)
		}
		bootArgs := req.BootArgs
		if bootArgs == "" {
			bootArgs = "console=ttyS0 reboot=k panic=1 pci=off"
		}
		bootArgs = networkBootArgs(bootArgs, networkConfig)
		if err := client.SetBootSource(ctx, firecracker.BootSource{KernelImagePath: req.KernelPath, BootArgs: bootArgs}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure boot source: %w", err)
		}
		if err := client.SetNetworkInterface(ctx, firecracker.NetworkInterface{IfaceID: networkConfig.IfaceID, HostDevName: networkConfig.TapDevice, GuestMAC: networkConfig.GuestMac}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure network interface: %w", err)
		}
		if err := client.SetVsock(ctx, firecracker.Vsock{GuestCID: s.guestAgentCID, HostPath: filepath.Join(filepath.Dir(socket), "vsock.sock")}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure vsock: %w", err)
		}
		if err := client.SetDrive(ctx, firecracker.Drive{DriveID: "rootfs", PathOnHost: req.RootfsPath, IsRootDevice: true, IsReadOnly: false}); err != nil {
			_ = proc.Process.Kill()
			return VM{}, fmt.Errorf("configure rootfs: %w", err)
		}
		if len(req.PortMappings) > 0 {
			if err := s.network.ApplyPortMappings(ctx, strings.Split(networkConfig.GuestCIDR, "/")[0], req.PortMappings); err != nil {
				_ = proc.Process.Kill()
				_ = s.network.Teardown(context.WithoutCancel(ctx), id)
				return VM{}, fmt.Errorf("configure port mappings: %w", err)
			}
		}
		vm.Logs = append(vm.Logs, "Firecracker process launched", "kernel, rootfs, TAP network, and port mappings configured")
		s.mu.Lock()
		s.processes[id] = proc
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.vms[id] = vm
	s.clients[id] = client
	s.mu.Unlock()
	if err := s.persist(); err != nil {
		return VM{}, fmt.Errorf("persist VM state: %w", err)
	}
	return vm, nil
}

func (s *Service) Start(ctx context.Context, id string) (VM, error) {
	client, vm, err := s.lookup(id)
	if err != nil {
		return VM{}, err
	}
	if vm.State != "created" && vm.State != "stopped" {
		return VM{}, fmt.Errorf("cannot start VM in state %q", vm.State)
	}
	if err := client.Start(ctx); err != nil {
		return s.appendLog(id, vm, "start failed: "+err.Error()), err
	}
	updated := s.appendLog(id, s.update(id, "running", vm), "microVM started")
	return updated, s.persist()
}

func (s *Service) Stop(ctx context.Context, id string) (VM, error) {
	client, vm, err := s.lookup(id)
	if err != nil {
		return VM{}, err
	}
	if vm.State != "running" && vm.State != "paused" {
		return VM{}, fmt.Errorf("cannot stop VM in state %q", vm.State)
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
		current, ok := s.vms[id]
		if ok {
			current.State = "stopped"
			current.UpdatedAt = time.Now().UTC()
			s.vms[id] = current
		}
		s.mu.Unlock()
		_ = s.persist()
	}()
	return updated, s.persist()
}

func (s *Service) Pause(ctx context.Context, id string) (VM, error) {
	return s.changeState(ctx, id, "paused", "running", func(c *firecracker.Client) error { return c.Pause(ctx) })
}
func (s *Service) Resume(ctx context.Context, id string) (VM, error) {
	return s.changeState(ctx, id, "running", "paused", func(c *firecracker.Client) error { return c.Resume(ctx) })
}
func (s *Service) changeState(ctx context.Context, id, stateName, requiredState string, action func(*firecracker.Client) error) (VM, error) {
	client, vm, err := s.lookup(id)
	if err != nil {
		return VM{}, err
	}
	if vm.State != requiredState {
		return VM{}, fmt.Errorf("cannot change VM from state %q to %q", vm.State, stateName)
	}
	if err := action(client); err != nil {
		return s.appendLog(id, vm, "state change failed: "+err.Error()), err
	}
	updated := s.appendLog(id, s.update(id, stateName, vm), "microVM state is "+stateName)
	return updated, s.persist()
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
	if persistErr := s.persist(); err == nil {
		err = persistErr
	}
	return err
}
func (s *Service) DeleteSnapshot(id, snapshotPath, memPath string) error {
	if _, _, err := s.lookup(id); err != nil {
		return err
	}
	if snapshotPath == "" || memPath == "" {
		return fmt.Errorf("snapshot and memory paths are required")
	}
	for _, path := range []string{snapshotPath, memPath} {
		if filepath.IsAbs(path) == false {
			return fmt.Errorf("snapshot paths must be absolute")
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove snapshot file %s: %w", path, err)
		}
	}
	if vm, ok := s.Get(id); ok {
		_ = s.appendLog(id, vm, "snapshot files deleted")
	}
	return s.persist()
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
	if persistErr := s.persist(); err == nil {
		err = persistErr
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
	delete(s.processes, id)
	delete(s.clients, id)
	delete(s.vms, id)
	factory, isDir := s.factory.(DirectorySocketFactory)
	s.mu.Unlock()
	if s.network != nil && vm.GuestIP != "" {
		s.network.RemovePortMappings(context.Background(), strings.Split(vm.GuestIP, "/")[0], vm.PortMappings)
		_ = s.network.Teardown(context.Background(), id)
	}
	if isDir {
		if err := os.RemoveAll(filepath.Join(factory.Dir, id)); err != nil {
			return fmt.Errorf("remove VM runtime directory: %w", err)
		}
	}
	return s.persist()
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
	out := make([]VM, 0, len(s.vms))
	for _, vm := range s.vms {
		out = append(out, vm)
	}
	return out
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
func (s *Service) update(id, stateName string, vm VM) VM {
	vm.State = stateName
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
func (s *Service) persist() error {
	if s.store == nil {
		return nil
	}
	return s.store.Save(s.List())
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
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(socket); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("Firecracker socket did not appear: %s", socket)
		case <-ticker.C:
		}
	}
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
