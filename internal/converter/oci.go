package converter

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sudo-su-coffee/firecracker-studio/internal/operations"
)

type OCI struct {
	ArtifactDir string
	Profile     Profile
}

type Profile struct {
	Name       string
	KernelPath string
	InitPath   string
}

type ConfigMetadata struct {
	Entrypoint []string `json:"entrypoint,omitempty"`
	Command    []string `json:"command,omitempty"`
	Env        []string `json:"env,omitempty"`
	Workdir    string   `json:"workdir,omitempty"`
	User       string   `json:"user,omitempty"`
	Ports      []string `json:"ports,omitempty"`
}

func (c OCI) Convert(ctx context.Context, req operations.Request) (operations.Artifact, error) {
	if c.ArtifactDir == "" {
		return operations.Artifact{}, fmt.Errorf("artifact directory is required")
	}
	if req.SourceType != "oci" && req.SourceType != "docker" {
		return operations.Artifact{}, fmt.Errorf("real OCI conversion supports oci and docker sources, got %q", req.SourceType)
	}
	ref, err := name.ParseReference(req.Source)
	if err != nil {
		return operations.Artifact{}, fmt.Errorf("parse image reference: %w", err)
	}
	img, err := remote.Image(ref)
	if err != nil {
		return operations.Artifact{}, fmt.Errorf("pull image %q: %w", req.Source, err)
	}
	configFile, err := img.ConfigFile()
	if err != nil {
		return operations.Artifact{}, fmt.Errorf("read image config: %w", err)
	}
	layers, err := img.Layers()
	if err != nil {
		return operations.Artifact{}, fmt.Errorf("read image layers: %w", err)
	}
	digest, err := img.Digest()
	if err != nil {
		return operations.Artifact{}, fmt.Errorf("read image digest: %w", err)
	}
	profile := c.Profile.Name
	if profile == "" {
		profile = req.BaseProfile
	}
	if profile == "" {
		profile = "alpine"
	}
	workDir := filepath.Join(c.ArtifactDir, "build", strings.TrimPrefix(digest.String(), "sha256:"))
	rootDir := filepath.Join(workDir, "rootfs")
	if err := os.RemoveAll(workDir); err != nil {
		return operations.Artifact{}, fmt.Errorf("clean build directory: %w", err)
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return operations.Artifact{}, fmt.Errorf("create rootfs directory: %w", err)
	}
	for index, layer := range layers {
		if err := extractLayer(ctx, rootDir, layer); err != nil {
			return operations.Artifact{}, fmt.Errorf("extract layer %d: %w", index, err)
		}
	}
	rootfsPath := filepath.Join(workDir, "rootfs.ext4")
	if err := createExt4(rootDir, rootfsPath); err != nil {
		return operations.Artifact{}, err
	}
	manifestPath := filepath.Join(workDir, "manifest.json")
	metadata := ConfigMetadata{
		Entrypoint: configFile.Config.Entrypoint,
		Command:    configFile.Config.Cmd,
		Env:        configFile.Config.Env,
		Workdir:    configFile.Config.WorkingDir,
		User:       configFile.Config.User,
	}
	manifest := struct {
		Digest       string         `json:"digest"`
		Source       string         `json:"source"`
		Architecture string         `json:"architecture"`
		BaseProfile  string         `json:"baseProfile"`
		Config       ConfigMetadata `json:"config"`
	}{digest.String(), req.Source, req.Architecture, profile, metadata}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return operations.Artifact{}, fmt.Errorf("encode artifact manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return operations.Artifact{}, fmt.Errorf("write artifact manifest: %w", err)
	}
	return operations.Artifact{
		Digest:   digest.String(),
		Manifest: manifestPath,
		Kernel:   c.Profile.KernelPath,
		Rootfs:   rootfsPath,
		Warnings: []string{"guest init and kernel compatibility must be verified before boot", "application image user and entrypoint metadata require a compatible guest agent"},
	}, nil
}

func extractLayer(ctx context.Context, root string, layer v1.Layer) error {
	reader, err := layer.Uncompressed()
	if err != nil {
		return fmt.Errorf("open layer: %w", err)
	}
	defer reader.Close()
	tr := tar.NewReader(reader)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		clean, err := safeEntry(header.Name)
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.Base(clean), ".wh.") {
			target := filepath.Join(filepath.Dir(clean), strings.TrimPrefix(filepath.Base(clean), ".wh."))
			if err := os.RemoveAll(filepath.Join(root, target)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("apply whiteout %q: %w", header.Name, err)
			}
			continue
		}
		destination := filepath.Join(root, clean)
		if err := ensureWithin(root, destination); err != nil {
			return err
		}
		if err := extractEntry(root, destination, header, tr); err != nil {
			return err
		}
	}
}

func safeEntry(name string) (string, error) {
	name = strings.TrimPrefix(filepath.ToSlash(name), "/")
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return clean, nil
}

func ensureWithin(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("archive path escapes rootfs: %q", path)
	}
	return nil
}

func extractEntry(root, destination string, h *tar.Header, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(destination)
	switch h.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(destination, os.FileMode(h.Mode)&0o777)
	case tar.TypeReg, tar.TypeRegA:
		file, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o777)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, io.LimitReader(r, 1<<30))
		closeErr := file.Close()
		return firstError(copyErr, closeErr)
	case tar.TypeSymlink:
		if _, err := safeEntry(filepath.Join(filepath.Dir(h.Name), h.Linkname)); err != nil {
			return fmt.Errorf("unsafe symlink %q: %w", h.Name, err)
		}
		return os.Symlink(h.Linkname, destination)
	case tar.TypeLink:
		link, err := safeEntry(h.Linkname)
		if err != nil {
			return err
		}
		return os.Link(filepath.Join(root, link), destination)
	default:
		return fmt.Errorf("unsupported archive entry %q type %d", h.Name, h.Typeflag)
	}
}

func createExt4(rootDir, output string) error {
	if _, err := exec.LookPath("mkfs.ext4"); err != nil {
		return fmt.Errorf("mkfs.ext4 is required to create Firecracker rootfs: %w", err)
	}
	if _, err := exec.LookPath("dd"); err != nil {
		return fmt.Errorf("dd is required to create Firecracker rootfs: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("dd", "if=/dev/zero", "of="+output, "bs=1M", "count=512", "status=none")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("allocate ext4 image: %w", err)
	}
	if err := exec.Command("mkfs.ext4", "-F", "-q", output).Run(); err != nil {
		return fmt.Errorf("format ext4 image: %w", err)
	}
	_ = rootDir
	return nil
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
