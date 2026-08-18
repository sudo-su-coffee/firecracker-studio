package converter

import (
	"context"

	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
)

type Hybrid struct {
	OCI    OCI
	DryRun DryRun
}

func (h Hybrid) Convert(ctx context.Context, req operations.Request) (operations.Artifact, error) {
	switch req.SourceType {
	case "oci", "docker":
		return h.OCI.Convert(ctx, req)
	default:
		return h.DryRun.Convert(ctx, req)
	}
}
