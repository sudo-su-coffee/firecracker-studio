package runtime

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const FirecrackerVersion = "v1.16.1"

var releaseChecksums = map[string]string{
	"x86_64":  "382a02a869e4d6d5cb14c40577f9545e8458021ea8b0b2d3fc10ec14d9c242e6",
	"aarch64": "8d0e69f6d6f9a1724551f607f18504052c16c1828ee3d4d7b6e6c73380871e0e",
}

type Status struct {
	Platform    string `json:"platform"`
	Installed   bool   `json:"installed"`
	Firecracker string `json:"firecracker"`
	Jailer      string `json:"jailer"`
	KVM         string `json:"kvm"`
	TAP         string `json:"tap"`
	Kernel      string `json:"kernel"`
	Rootfs      string `json:"rootfs"`
	Message     string `json:"message"`
}

type Manager struct {
	root string
	http *http.Client
}

func NewManager() *Manager {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return &Manager{root: filepath.Join(base, "FirecrackerStudio", "runtime"), http: &http.Client{Timeout: 10 * time.Minute}}
}

func (m *Manager) Root() string { return m.root }

func (m *Manager) Status(ctx context.Context) Status {
	status := Status{Platform: runtime.GOOS}
	if runtime.GOOS == "windows" {
		return m.wslStatus(ctx, status)
	}
	status.Firecracker = filepath.Join(m.root, "bin", "firecracker")
	status.Jailer = filepath.Join(m.root, "bin", "jailer")
	status.Installed = executable(status.Firecracker) && executable(status.Jailer)
	status.KVM = deviceStatus("/dev/kvm")
	status.TAP = tapStatus()
	status.Kernel = artifactStatus(filepath.Join(m.root, "images", "default", "vmlinux"))
	status.Rootfs = artifactStatus(filepath.Join(m.root, "images", "default", "rootfs.ext4"))
	if status.Installed {
		status.Message = "Firecracker and jailer are installed"
	} else {
		status.Message = "Install the local Firecracker runtime to continue"
	}
	return status
}

func (m *Manager) Install(ctx context.Context) (Status, error) {
	if runtime.GOOS == "windows" {
		return m.installWSL(ctx)
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	if arch == "arm64" {
		arch = "aarch64"
	}
	expected, ok := releaseChecksums[arch]
	if !ok {
		return Status{}, fmt.Errorf("unsupported architecture %s", arch)
	}
	if err := os.MkdirAll(filepath.Join(m.root, "bin"), 0700); err != nil {
		return Status{}, err
	}
	url := fmt.Sprintf("https://github.com/firecracker-microvm/firecracker/releases/download/%s/firecracker-%s-%s.tgz", FirecrackerVersion, FirecrackerVersion, arch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Status{}, err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		return Status{}, fmt.Errorf("download Firecracker: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("download Firecracker returned HTTP %d", resp.StatusCode)
	}
	archivePath := filepath.Join(m.root, "firecracker-release.tgz")
	file, err := os.Create(archivePath)
	if err != nil {
		return Status{}, err
	}
	hash := sha256.New()
	if _, err = io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
		file.Close()
		return Status{}, err
	}
	if err = file.Close(); err != nil {
		return Status{}, err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); !strings.EqualFold(got, expected) {
		return Status{}, fmt.Errorf("Firecracker checksum mismatch: got %s", got)
	}
	if err := extractBinaries(archivePath, filepath.Join(m.root, "bin"), FirecrackerVersion, arch); err != nil {
		return Status{}, err
	}
	_ = os.Remove(archivePath)
	return m.Status(ctx), nil
}

func extractBinaries(archivePath, destination, version, arch string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tarReader := tar.NewReader(gz)
	wanted := map[string]string{"firecracker-" + version + "-" + arch: "firecracker", "jailer-" + version + "-" + arch: "jailer"}
	found := map[string]bool{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Base(header.Name)
		target, ok := wanted[name]
		if !ok || header.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.OpenFile(filepath.Join(destination, target), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
		if err != nil {
			return err
		}
		if _, err = io.Copy(out, tarReader); err != nil {
			out.Close()
			return err
		}
		if err = out.Close(); err != nil {
			return err
		}
		found[target] = true
	}
	if !found["firecracker"] || !found["jailer"] {
		return fmt.Errorf("official archive did not contain both firecracker and jailer")
	}
	return nil
}

func (m *Manager) wslStatus(ctx context.Context, status Status) Status {
	status.Message = "WSL2 local runtime is managed from the desktop app"
	check := exec.CommandContext(ctx, "wsl.exe", "--", "bash", "-lc", "command -v firecracker >/dev/null && command -v jailer >/dev/null && test -r /dev/kvm && test -w /dev/kvm")
	if err := check.Run(); err == nil {
		status.Installed = true
		status.KVM = "ready"
		status.Firecracker = "installed"
		status.Jailer = "installed"
	} else {
		status.KVM = "check WSL2"
		status.Firecracker = "not installed"
		status.Jailer = "not installed"
	}
	status.TAP = "worker-managed"
	status.Kernel = "catalog"
	status.Rootfs = "catalog"
	return status
}

func (m *Manager) installWSL(ctx context.Context) (Status, error) {
	arch := "x86_64"
	if out, err := exec.CommandContext(ctx, "wsl.exe", "--", "uname", "-m").Output(); err == nil && strings.Contains(string(out), "aarch64") {
		arch = "aarch64"
	}
	expected := releaseChecksums[arch]
	root := "$HOME/.config/firecracker-studio/runtime"
	url := fmt.Sprintf("https://github.com/firecracker-microvm/firecracker/releases/download/%s/firecracker-%s-%s.tgz", FirecrackerVersion, FirecrackerVersion, arch)
	script := fmt.Sprintf("set -e; mkdir -p %s/bin; tmp=$(mktemp); work=$(mktemp -d); trap 'rm -rf $tmp $work' EXIT; curl -fsSL '%s' -o $tmp; echo '%s  '$tmp | sha256sum -c -; tar -xzf $tmp -C $work; fc=$(find $work -type f -name 'firecracker-%s-%s' -print -quit); jailer=$(find $work -type f -name 'jailer-%s-%s' -print -quit); test -n \"$fc\"; test -n \"$jailer\"; install -m 700 \"$fc\" %s/bin/firecracker; install -m 700 \"$jailer\" %s/bin/jailer", root, url, expected, FirecrackerVersion, arch, FirecrackerVersion, arch, root, root)
	if out, err := exec.CommandContext(ctx, "wsl.exe", "--", "bash", "-lc", script).CombinedOutput(); err != nil {
		return m.Status(ctx), fmt.Errorf("WSL2 runtime install failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return m.Status(ctx), nil
}

func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().Perm()&0111 != 0
}
func artifactStatus(path string) string {
	if _, err := os.Stat(path); err == nil {
		return "present"
	}
	return "missing"
}
func deviceStatus(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	if info.Mode().Perm()&0600 == 0600 {
		return "ready"
	}
	return "permission denied"
}
func tapStatus() string {
	if _, err := exec.LookPath("ip"); err != nil {
		return "install iproute2"
	}
	return "check worker networking"
}
