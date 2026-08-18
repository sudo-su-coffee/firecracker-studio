package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	Smt        bool `json:"smt"`
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

type SnapshotCreate struct {
	SnapshotType string `json:"snapshot_type"`
	SnapshotPath string `json:"snapshot_path"`
	MemFilePath  string `json:"mem_file_path"`
}

type SnapshotLoad struct {
	SnapshotPath string `json:"snapshot_path"`
	MemBackend   string `json:"mem_backend"`
	EnableDiff   bool   `json:"enable_diff_snapshots"`
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

func (c *Client) Start(ctx context.Context) error {
	return c.put(ctx, "/actions", map[string]string{"action_type": "InstanceStart"})
}

func (c *Client) SendCtrlAltDel(ctx context.Context) error {
	return c.put(ctx, "/actions", map[string]string{"action_type": "SendCtrlAltDel"})
}

func (c *Client) CreateSnapshot(ctx context.Context, snapshot SnapshotCreate) error {
	return c.put(ctx, "/snapshot/create", snapshot)
}

func (c *Client) LoadSnapshot(ctx context.Context, snapshot SnapshotLoad) error {
	return c.put(ctx, "/snapshot/load", snapshot)
}

func (c *Client) put(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Firecracker request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://localhost"+path, bytes.NewReader(body))
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
		return fmt.Errorf("Firecracker request %s returned HTTP %d", path, resp.StatusCode)
	}
	return nil
}
