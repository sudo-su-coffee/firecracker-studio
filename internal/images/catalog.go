package images

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sudo-su-coffee/firecracker-studio/internal/state"
)

type Image struct {
	ID            string    `json:"id"`
	Name          string    `json:"name,omitempty"`
	Tag           string    `json:"tag,omitempty"`
	Reference     string    `json:"reference"`
	SourceType    string    `json:"sourceType"`
	SourceDigest  string    `json:"sourceDigest,omitempty"`
	Digest        string    `json:"digest"`
	Architecture  string    `json:"architecture"`
	BaseProfile   string    `json:"baseProfile"`
	KernelPath    string    `json:"kernelPath,omitempty"`
	RootfsPath    string    `json:"rootfsPath,omitempty"`
	ArtifactPath  string    `json:"artifactPath,omitempty"`
	Command       []string  `json:"command,omitempty"`
	Environment   []string  `json:"environment,omitempty"`
	GuestAgent    bool      `json:"guestAgent"`
	SerialConsole bool      `json:"serialConsole"`
	Status        string    `json:"status"`
	Error         string    `json:"error,omitempty"`
	SizeBytes     int64     `json:"sizeBytes"`
	Verified      bool      `json:"verified"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Catalog struct {
	mu     sync.RWMutex
	images map[string]Image
	store  *state.Store[Image]
}

func NewCatalog(path string) (*Catalog, error) {
	store, err := state.New[Image](path)
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{images: make(map[string]Image), store: store}
	items, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("load image catalog: %w", err)
	}
	for _, image := range items {
		catalog.images[image.Digest] = image
	}
	return catalog, nil
}

func (c *Catalog) Upsert(image Image) (Image, error) {
	if strings.TrimSpace(image.Reference) == "" {
		return Image{}, fmt.Errorf("image reference is required")
	}
	if strings.TrimSpace(image.Digest) == "" {
		return Image{}, fmt.Errorf("image digest is required")
	}
	if image.SourceType == "" {
		image.SourceType = "oci"
	}
	if image.BaseProfile == "" {
		image.BaseProfile = "alpine"
	}
	if image.Architecture == "" {
		image.Architecture = "native"
	}
	if image.Status == "" {
		image.Status = "ready"
	}
	if image.Name == "" {
		image.Name = image.Reference
	}
	if image.RootfsPath == "" {
		image.RootfsPath = image.ArtifactPath
	}
	if image.ArtifactPath == "" {
		image.ArtifactPath = image.RootfsPath
	}
	now := time.Now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.images[image.Digest]; ok {
		image.CreatedAt = existing.CreatedAt
	}
	if image.CreatedAt.IsZero() {
		image.CreatedAt = now
	}
	image.UpdatedAt = now
	c.images[image.Digest] = image
	if err := c.saveLocked(); err != nil {
		return Image{}, err
	}
	return image, nil
}

func (c *Catalog) Get(digest string) (Image, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	image, ok := c.images[digest]
	return image, ok
}

func (c *Catalog) Delete(digest string, removeArtifacts bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	image, ok := c.images[digest]
	if !ok {
		return fmt.Errorf("image %q not found", digest)
	}
	if removeArtifacts {
		for _, path := range []string{image.KernelPath, image.RootfsPath, image.ArtifactPath} {
			if path != "" {
				_ = os.Remove(path)
			}
		}
	}
	delete(c.images, digest)
	return c.saveLocked()
}

func (c *Catalog) StorageBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var total int64
	for _, image := range c.images {
		total += image.SizeBytes
	}
	return total
}

func (c *Catalog) List() []Image {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := make([]Image, 0, len(c.images))
	for _, image := range c.images {
		items = append(items, image)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items
}

func (c *Catalog) saveLocked() error {
	items := make([]Image, 0, len(c.images))
	for _, image := range c.images {
		items = append(items, image)
	}
	return c.store.Save(items)
}

func ArtifactSize(image Image) int64 {
	var total int64
	for _, path := range []string{image.KernelPath, image.RootfsPath, image.ArtifactPath} {
		if path == "" || (image.RootfsPath != "" && path == image.ArtifactPath) {
			continue
		}
		if info, err := os.Stat(filepath.Clean(path)); err == nil {
			total += info.Size()
		}
	}
	return total
}
