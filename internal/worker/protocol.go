package worker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const ProtocolVersion = "v1"

type Capabilities struct {
	WorkerID        string   `json:"workerId"`
	ProtocolVersion string   `json:"protocolVersion"`
	OS              string   `json:"os"`
	Architecture    string   `json:"architecture"`
	Firecracker     string   `json:"firecracker"`
	KVM             bool     `json:"kvm"`
	TAP             bool     `json:"tap"`
	Cgroups         bool     `json:"cgroups"`
	Jailer          bool     `json:"jailer"`
	KernelProfiles  []string `json:"kernelProfiles"`
	RootfsTools     bool     `json:"rootfsTools"`
}

type VMRequest struct {
	ArtifactDigest string        `json:"artifactDigest"`
	ImageReference string        `json:"imageReference,omitempty"`
	KernelPath     string        `json:"kernelPath,omitempty"`
	RootfsPath     string        `json:"rootfsPath,omitempty"`
	BootArgs       string        `json:"bootArgs,omitempty"`
	VCPUs          int           `json:"vcpus"`
	MemoryMiB      int           `json:"memoryMiB"`
	PortMappings   []PortMapping `json:"portMappings,omitempty"`
}

type PortMapping struct {
	HostPort  int    `json:"hostPort"`
	GuestPort int    `json:"guestPort"`
	Protocol  string `json:"protocol"`
}

type VM struct {
	ID             string        `json:"id"`
	State          string        `json:"state"`
	ArtifactDigest string        `json:"artifactDigest"`
	ImageReference string        `json:"imageReference,omitempty"`
	KernelPath     string        `json:"kernelPath,omitempty"`
	RootfsPath     string        `json:"rootfsPath,omitempty"`
	PortMappings   []PortMapping `json:"portMappings,omitempty"`
	Logs           []string      `json:"logs,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL, bearerToken, certificatePEM string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("worker URL is required")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if certificatePEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(certificatePEM)) {
			return nil, fmt.Errorf("invalid worker certificate PEM")
		}
		transport.TLSClientConfig.RootCAs = pool
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout, Transport: bearerTransport{base: transport, token: bearerToken}}}, nil
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if t.token != "" {
		clone.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(clone)
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	var out Capabilities
	if err := c.do(ctx, http.MethodGet, "/api/v1/capabilities", nil, &out); err != nil {
		return Capabilities{}, err
	}
	return out, nil
}

func (c *Client) CreateVM(ctx context.Context, req VMRequest) (VM, error) {
	var out VM
	if err := c.do(ctx, http.MethodPost, "/api/v1/vms", req, &out); err != nil {
		return VM{}, err
	}
	return out, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode worker request: %w", err)
		}
		reader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create worker request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("worker request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("worker returned HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode worker response: %w", err)
	}
	return nil
}
