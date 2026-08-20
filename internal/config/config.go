package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type AdminConfig struct {
	Username     string `toml:"username" json:"username"`
	Email        string `toml:"email" json:"email"`
	PasswordHash string `toml:"password_hash" json:"-"`
}

type NotificationConfig struct {
	Enabled          bool     `toml:"enabled" json:"enabled"`
	SMTPHost         string   `toml:"smtp_host" json:"smtpHost"`
	SMTPPort         int      `toml:"smtp_port" json:"smtpPort"`
	SMTPUsername     string   `toml:"smtp_username" json:"smtpUsername"`
	SMTPPassword     string   `toml:"smtp_password" json:"-"`
	SMTPPasswordFile string   `toml:"smtp_password_file" json:"-"`
	From             string   `toml:"from" json:"from"`
	Recipients       []string `toml:"recipients" json:"recipients"`
}

type Config struct {
	AppName            string             `toml:"app_name" json:"appName"`
	ListenAddress      string             `toml:"listen" json:"listenAddress"`
	PublicURL          string             `toml:"public_url" json:"publicUrl"`
	StateDir           string             `toml:"state_dir" json:"stateDir"`
	ArtifactDir        string             `toml:"artifact_dir" json:"artifactDir"`
	OperationRetention time.Duration      `toml:"operation_retention" json:"operationRetention"`
	WorkerTimeout      time.Duration      `toml:"worker_timeout" json:"workerTimeout"`
	OperationWorkers   int                `toml:"operation_workers" json:"operationWorkers"`
	Admin              AdminConfig        `toml:"admin" json:"admin"`
	Notifications      NotificationConfig `toml:"notifications" json:"notifications"`
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
		PublicURL:          "http://127.0.0.1:7822",
		StateDir:           home,
		ArtifactDir:        filepath.Join(home, "artifacts"),
		OperationRetention: 24 * time.Hour,
		WorkerTimeout:      30 * time.Second,
		OperationWorkers:   2,
		Admin:              AdminConfig{Username: "admin"},
		Notifications:      NotificationConfig{SMTPPort: 587},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if strings.TrimSpace(path) == "" {
		return applyEnvironment(cfg)
	}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config.toml: %w", err)
	}
	defer file.Close()
	if err := toml.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config.toml: %w", err)
	}
	return applyEnvironment(cfg)
}

func applyEnvironment(cfg Config) (Config, error) {
	if address := os.Getenv("FIRECRACKER_STUDIO_LISTEN"); address != "" {
		cfg.ListenAddress = address
	}
	if publicURL := os.Getenv("FIRECRACKER_STUDIO_PUBLIC_URL"); publicURL != "" {
		cfg.PublicURL = publicURL
	}
	return cfg, cfg.Validate()
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
	if c.Admin.Username == "" {
		return fmt.Errorf("admin username is required")
	}
	if !loopbackAddress(c.ListenAddress) && strings.TrimSpace(os.Getenv("FIRECRACKER_STUDIO_TOKEN")) == "" && c.Admin.PasswordHash == "" {
		return fmt.Errorf("non-loopback listen address requires admin password_hash or FIRECRACKER_STUDIO_TOKEN")
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
