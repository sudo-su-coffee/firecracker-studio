package images

import "time"

type BaseImage struct {
	ID            string     `json:"id"`
	Distribution  string     `json:"distribution"`
	Version       string     `json:"version"`
	Architecture  string     `json:"architecture"`
	KernelChannel string     `json:"kernelChannel"`
	KernelPath    string     `json:"kernelPath,omitempty"`
	RootfsPath    string     `json:"rootfsPath,omitempty"`
	RootfsFormat  string     `json:"rootfsFormat"`
	InitSystem    string     `json:"initSystem"`
	SourceURL     string     `json:"sourceUrl,omitempty"`
	Checksum      string     `json:"checksum,omitempty"`
	Status        string     `json:"status"`
	Default       bool       `json:"default"`
	DownloadedAt  *time.Time `json:"downloadedAt,omitempty"`
}

func ManagedBaseImages() []BaseImage {
	return []BaseImage{
		{ID: "alpine-3.24.1-x86_64", Distribution: "alpine", Version: "3.24.1", Architecture: "x86_64", KernelChannel: "6.1", RootfsFormat: "ext4", InitSystem: "openrc", Status: "catalog", Default: true},
		{ID: "alpine-3.24.1-aarch64", Distribution: "alpine", Version: "3.24.1", Architecture: "aarch64", KernelChannel: "6.1", RootfsFormat: "ext4", InitSystem: "openrc", Status: "catalog"},
		{ID: "debian-12-x86_64", Distribution: "debian", Version: "12", Architecture: "x86_64", KernelChannel: "6.1", RootfsFormat: "ext4", InitSystem: "systemd", Status: "catalog"},
		{ID: "debian-12-aarch64", Distribution: "debian", Version: "12", Architecture: "aarch64", KernelChannel: "6.1", RootfsFormat: "ext4", InitSystem: "systemd", Status: "catalog"},
		{ID: "ubuntu-22.04-x86_64", Distribution: "ubuntu", Version: "22.04", Architecture: "x86_64", KernelChannel: "6.1", RootfsFormat: "ext4", InitSystem: "systemd", Status: "catalog"},
		{ID: "ubuntu-22.04-aarch64", Distribution: "ubuntu", Version: "22.04", Architecture: "aarch64", KernelChannel: "6.1", RootfsFormat: "ext4", InitSystem: "systemd", Status: "catalog"},
	}
}

func ManagedBaseImage(id string) (BaseImage, bool) {
	for _, image := range ManagedBaseImages() {
		if image.ID == id {
			return image, true
		}
	}
	return BaseImage{}, false
}
