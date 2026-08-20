package images

import (
	"path/filepath"
	"testing"
)

func TestCatalogPersistsImages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "images.json")
	catalog, err := NewCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Upsert(Image{Reference: "alpine:3.20", Digest: "sha256:test", SourceType: "docker", Status: "ready"}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	image, ok := reloaded.Get("sha256:test")
	if !ok || image.Reference != "alpine:3.20" || image.Status != "ready" {
		t.Fatalf("unexpected reloaded image: %+v", image)
	}
}

func TestCatalogDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "images.json")
	catalog, err := NewCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Upsert(Image{Reference: "app:latest", Digest: "sha256:app"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Delete("sha256:app", false); err != nil {
		t.Fatal(err)
	}
	if _, ok := catalog.Get("sha256:app"); ok {
		t.Fatal("image should be deleted")
	}
}
