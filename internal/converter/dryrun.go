package converter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
)

type DryRun struct{}

func (DryRun) Convert(_ context.Context, req operations.Request) (operations.Artifact, error) {
	if strings.TrimSpace(req.Source) == "" {
		return operations.Artifact{}, fmt.Errorf("image source is empty")
	}
	if req.SourceType != "oci" && req.SourceType != "docker" && req.SourceType != "dockerfile" && req.SourceType != "archive" {
		return operations.Artifact{}, fmt.Errorf("unsupported source type %q", req.SourceType)
	}
	profile := req.BaseProfile
	if profile == "" {
		profile = "alpine"
	}
	hash := sha256.Sum256([]byte(req.SourceType + "\x00" + req.Source + "\x00" + profile + "\x00" + req.Architecture))
	digest := "sha256:" + hex.EncodeToString(hash[:])
	return operations.Artifact{
		Digest:   digest,
		Manifest: "pending-oci-conversion",
		Kernel:   "pending-kernel-profile/" + profile,
		Rootfs:   "pending-rootfs/" + digest,
		Warnings: []string{"v0.1.0 dry-run converter: OCI layer extraction and Firecracker rootfs generation are not yet enabled"},
	}, nil
}
