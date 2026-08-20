package resources

// State is the durable aggregate for server-managed Firecracker resources.
// Maps are persisted as JSON objects by the generic atomic state store.
type State struct {
	MachineConfigs map[string]MachineConfig `json:"machineConfigs"`
	Kernels        map[string]Kernel        `json:"kernels"`
	Volumes        map[string]Volume        `json:"volumes"`
	Vsocks         map[string]VsockConfig   `json:"vsocks"`
}

func NewState() State {
	return State{
		MachineConfigs: map[string]MachineConfig{},
		Kernels:        map[string]Kernel{},
		Volumes:        map[string]Volume{},
		Vsocks:         map[string]VsockConfig{},
	}
}
