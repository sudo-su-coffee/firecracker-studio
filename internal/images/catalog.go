package images

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Image struct {
	ID           string    `json:"id"`
	Reference    string    `json:"reference"`
	SourceType   string    `json:"sourceType"`
	Digest       string    `json:"digest"`
	Architecture string    `json:"architecture"`
	BaseProfile  string    `json:"baseProfile"`
	ArtifactPath string    `json:"artifactPath,omitempty"`
	Verified     bool      `json:"verified"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Catalog struct {
	mu     sync.RWMutex
	images map[string]Image
}

func NewCatalog() *Catalog {
	return &Catalog{images: make(map[string]Image)}
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
	return image, nil
}

func (c *Catalog) Get(digest string) (Image, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	image, ok := c.images[digest]
	return image, ok
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
