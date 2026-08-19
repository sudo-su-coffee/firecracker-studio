package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	AppName            string        `json:"appName"`
	ListenAddress      string        `json:"listenAddress"`
	StateDir           string        `json:"stateDir"`
	ArtifactDir        string        `json:"artifactDir"`
	OperationRetention time.Duration `json:"operationRetention"`
	WorkerTimeout      time.Duration `json:"workerTimeout"`
	OperationWorkers   int           `json:"operationWorkers"`
}

func Default() Config {
	home, err := os.UserConfigDir()
	if err != nil || home == "" {
		home = ".firecracker-studio"
	} else {
		home = filepath.Join(home, "FirecrackerStudio")
	}
	return Config{
		AppName:            "firecracker-studio",
		ListenAddress:      "127.0.0.1:7822",
		StateDir:           home,
		ArtifactDir:        filepath.Join(home, "artifacts"),
		OperationRetention: 24 * time.Hour,
		WorkerTimeout:      30 * time.Second,
		OperationWorkers:   2,
	}
}

func (c Config) Validate() error {
	if c.AppName == "" {
		return fmt.Errorf("app name is required")
	}
	if c.ListenAddress == "" {
		return fmt.Errorf("listen address is required")
	}
	if c.StateDir == "" || c.ArtifactDir == "" {
		return fmt.Errorf("state and artifact directories are required")
	}
	if c.OperationRetention <= 0 || c.WorkerTimeout <= 0 {
		return fmt.Errorf("retention and worker timeout must be positive")
	}
	if c.OperationWorkers < 1 || c.OperationWorkers > 128 {
		return fmt.Errorf("operation workers must be between 1 and 128")
	}
	if !loopbackAddress(c.ListenAddress) && strings.TrimSpace(os.Getenv("FIRECRACKER_STUDIO_TOKEN")) == "" {
		return fmt.Errorf("non-loopback listen address requires FIRECRACKER_STUDIO_TOKEN")
	}
	return nil
}

func loopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address == "localhost"
	}
	if host == "localhost" || host == "" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
