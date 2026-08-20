package resources

import (
	"fmt"
	"strings"
	"time"
)

type MachineConfig struct {
	VCPUs      int       `json:"vcpus"`
	MemoryMiB  int       `json:"memoryMiB"`
	CPUModel   string    `json:"cpuModel,omitempty"`
	SMT        bool      `json:"smt"`
	BootArgs   string    `json:"bootArgs,omitempty"`
	KernelPath string    `json:"kernelPath,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Kernel struct {
	ID, Path, Architecture, Version, Digest string
	RegisteredAt                            time.Time `json:"registeredAt"`
}
type Drive struct {
	ID, Path, Kind       string
	ReadOnly, Persistent bool
	SizeBytes            int64
	AttachedVM           string `json:"attachedVm,omitempty"`
}
type Volume struct {
	ID, Path, Filesystem string
	SizeBytes, UsedBytes int64
	Persistent           bool
	AttachedVM           string    `json:"attachedVm,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
}
type NetworkInterface struct {
	ID, TapDevice, GuestIP, HostIP, GuestMAC, Mode string
	HostPort, GuestPort                            int
	Protocol                                       string
}
type VsockConfig struct {
	GuestCID                uint32
	HostPath, AgentPort     string
	Enabled, AgentAvailable bool
}
type MetricsConfig struct {
	Path, Format string
	Enabled      bool
}
type LoggerConfig struct {
	Path, Level string
	Enabled     bool
}
type BalloonConfig struct {
	Enabled              bool
	TargetMiB, ActualMiB int
}

func ValidateMachine(c MachineConfig) error {
	if c.VCPUs < 1 || c.VCPUs > 256 {
		return fmt.Errorf("vcpus must be between 1 and 256")
	}
	if c.MemoryMiB < 128 {
		return fmt.Errorf("memoryMiB must be at least 128")
	}
	return nil
}
func ValidateStorageMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "ephemeral" && mode != "persistent" {
		return fmt.Errorf("storage mode must be ephemeral or persistent")
	}
	return nil
}
func ValidatePort(host, guest int) error {
	if host < 1 || host > 65535 || guest < 1 || guest > 65535 {
		return fmt.Errorf("ports must be between 1 and 65535")
	}
	return nil
}
