package images

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MarketplaceCatalog struct {
	SchemaVersion int           `json:"schemaVersion"`
	Repository    string        `json:"repository"`
	Images        []CatalogItem `json:"images"`
}

type CatalogItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Distribution string `json:"distribution"`
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	RootfsFormat string `json:"rootfsFormat"`
	Status       string `json:"status"`
	Default      bool   `json:"default"`
	Description  string `json:"description"`
	KernelURL    string `json:"kernelUrl"`
	RootfsURL    string `json:"rootfsUrl"`
	ChecksumURL  string `json:"checksumUrl"`
	KernelSHA256 string `json:"kernelSha256"`
	RootfsSHA256 string `json:"rootfsSha256"`
}

type Marketplace struct {
	CatalogURL string
	Root       string
	Client     *http.Client
}

func NewMarketplace(root, catalogURL string) *Marketplace {
	if catalogURL == "" {
		catalogURL = "https://raw.githubusercontent.com/sudo-su-coffee/firecracker-marketplace/main/catalog.json"
	}
	return &Marketplace{CatalogURL: catalogURL, Root: root, Client: &http.Client{Timeout: 30 * time.Minute}}
}

func (m *Marketplace) List(ctx context.Context) ([]CatalogItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.CatalogURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("marketplace catalog returned HTTP %d", resp.StatusCode)
	}
	var catalog MarketplaceCatalog
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("decode marketplace catalog: %w", err)
	}
	return catalog.Images, nil
}

func (m *Marketplace) Pull(ctx context.Context, id string) (CatalogItem, string, error) {
	items, err := m.List(ctx)
	if err != nil {
		return CatalogItem{}, "", err
	}
	var item CatalogItem
	for _, candidate := range items {
		if candidate.ID == id {
			item = candidate
			break
		}
	}
	if item.ID == "" {
		return CatalogItem{}, "", fmt.Errorf("marketplace image %q not found", id)
	}
	if item.KernelURL == "" || item.RootfsURL == "" {
		return CatalogItem{}, "", fmt.Errorf("image %q has no verified downloadable kernel and rootfs", id)
	}
	dir := filepath.Join(m.Root, "images", id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return CatalogItem{}, "", err
	}
	kernel := filepath.Join(dir, "vmlinux")
	rootfs := filepath.Join(dir, "rootfs."+item.RootfsFormat)
	if err := m.download(ctx, item.KernelURL, kernel, item.KernelSHA256); err != nil {
		return CatalogItem{}, "", fmt.Errorf("download kernel: %w", err)
	}
	if err := m.download(ctx, item.RootfsURL, rootfs, item.RootfsSHA256); err != nil {
		return CatalogItem{}, "", fmt.Errorf("download rootfs: %w", err)
	}
	return item, dir, nil
}

func (m *Marketplace) download(ctx context.Context, source, destination, expected string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	resp, err := m.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("source returned HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".download-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hash), resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if expected != "" {
		got := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(strings.TrimSpace(expected), got) {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, got)
		}
	}
	return os.Rename(tmp.Name(), destination)
}
