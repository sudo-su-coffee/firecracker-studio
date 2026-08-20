package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Client struct {
	socket string
	http   *http.Client
}

type MachineConfig struct {
	VCPUCount  int  `json:"vcpu_count"`
	MemSizeMiB int  `json:"mem_size_mib"`
	Smt        bool `json:"smt,omitempty"`
}

type BootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args"`
}

type Drive struct {
	DriveID      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}

type NetworkInterface struct {
	IfaceID     string `json:"iface_id"`
	HostDevName string `json:"host_dev_name"`
	GuestMAC    string `json:"guest_mac,omitempty"`
}

type Vsock struct {
	GuestCID uint32 `json:"guest_cid"`
	HostPath string `json:"uds_path"`
}

type MMDSConfig struct {
	NetworkInterfaces []string `json:"network_interfaces,omitempty"`
	Version           string   `json:"version,omitempty"`
}

type VMState struct {
	State string `json:"state"`
}

type SnapshotCreate struct {
	SnapshotType string `json:"snapshot_type"`
	SnapshotPath string `json:"snapshot_path"`
	MemFilePath  string `json:"mem_file_path"`
}

type SnapshotLoad struct {
	SnapshotPath string `json:"snapshot_path"`
	MemBackend   string `json:"mem_backend"`
	EnableDiff   bool   `json:"enable_diff_snapshots,omitempty"`
}

func NewClient(socket string, timeout time.Duration) (*Client, error) {
	if socket == "" {
		return nil, fmt.Errorf("Firecracker socket is required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socket)
		},
	}
	return &Client{socket: socket, http: &http.Client{Transport: transport, Timeout: timeout}}, nil
}

func (c *Client) SetMachineConfig(ctx context.Context, config MachineConfig) error {
	return c.put(ctx, "/machine-config", config)
}
func (c *Client) SetBootSource(ctx context.Context, boot BootSource) error {
	return c.put(ctx, "/boot-source", boot)
}
func (c *Client) SetDrive(ctx context.Context, drive Drive) error {
	return c.put(ctx, "/drives/"+drive.DriveID, drive)
}
func (c *Client) SetNetworkInterface(ctx context.Context, nic NetworkInterface) error {
	return c.put(ctx, "/network-interfaces/"+nic.IfaceID, nic)
}
func (c *Client) SetVsock(ctx context.Context, vsock Vsock) error {
	return c.put(ctx, "/vsocks", vsock)
}
func (c *Client) SetMMDSConfig(ctx context.Context, config MMDSConfig) error {
	return c.put(ctx, "/mmds/config", config)
}
func (c *Client) PutMMDS(ctx context.Context, data any) error { return c.put(ctx, "/mmds", data) }
func (c *Client) SetMetrics(ctx context.Context, path string) error {
	return c.put(ctx, "/metrics", map[string]string{"metrics_path": path})
}
func (c *Client) SetLogger(ctx context.Context, path string) error {
	return c.put(ctx, "/logger", map[string]any{"log_path": path, "level": "Info", "show_level": true, "show_log_origin": true})
}
func (c *Client) Start(ctx context.Context) error {
	return c.put(ctx, "/actions", map[string]string{"action_type": "InstanceStart"})
}
func (c *Client) SendCtrlAltDel(ctx context.Context) error {
	return c.put(ctx, "/actions", map[string]string{"action_type": "SendCtrlAltDel"})
}
func (c *Client) Pause(ctx context.Context) error {
	return c.patch(ctx, "/vm", VMState{State: "Paused"})
}
func (c *Client) Resume(ctx context.Context) error {
	return c.patch(ctx, "/vm", VMState{State: "Resumed"})
}
func (c *Client) State(ctx context.Context) (VMState, error) {
	var state VMState
	err := c.get(ctx, "/vm", &state)
	return state, err
}
func (c *Client) CreateSnapshot(ctx context.Context, snapshot SnapshotCreate) error {
	return c.put(ctx, "/snapshot/create", snapshot)
}
func (c *Client) LoadSnapshot(ctx context.Context, snapshot SnapshotLoad) error {
	return c.put(ctx, "/snapshot/load", snapshot)
}

func (c *Client) OpenVsock(ctx context.Context, port uint32) (net.Conn, error) {
	dialer := net.Dialer{Timeout: c.http.Timeout}
	conn, err := dialer.DialContext(ctx, "unix", c.socket+".vsock")
	if err != nil {
		return nil, fmt.Errorf("connect vsock proxy: %w", err)
	}
	if _, err := fmt.Fprintf(conn, "CONNECT %d\n", port); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("vsock handshake: %w", err)
	}
	return conn, nil
}

func (c *Client) put(ctx context.Context, path string, payload any) error {
	return c.request(ctx, http.MethodPut, path, payload, nil)
}
func (c *Client) patch(ctx context.Context, path string, payload any) error {
	return c.request(ctx, http.MethodPatch, path, payload, nil)
}
func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.request(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) request(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode Firecracker request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://localhost"+path, body)
	if err != nil {
		return fmt.Errorf("create Firecracker request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Firecracker request %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Firecracker request %s returned HTTP %d: %s", path, resp.StatusCode, string(message))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode Firecracker response %s: %w", path, err)
		}
	}
	return nil
}
