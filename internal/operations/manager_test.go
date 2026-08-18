package operations

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

type testConverter struct {
	err error
}

func (c testConverter) Convert(_ context.Context, req Request) (Artifact, error) {
	if c.err != nil {
		return Artifact{}, c.err
	}
	return Artifact{Digest: "sha256:test", Manifest: req.Source}, nil
}

func TestManagerEnqueueCompletes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := NewManager(ctx, 1, testConverter{}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	op, err := manager.Enqueue(ctx, Request{Source: "alpine:latest", SourceType: "oci"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		current, ok := manager.Get(op.ID)
		if ok && current.State == StateSucceeded {
			if current.Artifact == nil || current.Artifact.Digest != "sha256:test" {
				t.Fatalf("unexpected artifact: %#v", current.Artifact)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	current, _ := manager.Get(op.ID)
	t.Fatalf("operation %s did not complete: %#v", op.ID, current)
}

func TestManagerRecordsFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := NewManager(ctx, 1, testConverter{err: errors.New("conversion failed")}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	op, err := manager.Enqueue(ctx, Request{Source: "bad-image", SourceType: "oci"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		current, ok := manager.Get(op.ID)
		if ok && current.State == StateFailed {
			if current.Error != "conversion failed" {
				t.Fatalf("unexpected error: %q", current.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("operation %s did not fail", op.ID)
}
