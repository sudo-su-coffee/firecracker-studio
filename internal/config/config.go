package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	value := strings.TrimSpace(string(text))
	if parsed, err := time.ParseDuration(value); err == nil {
		*d = Duration(parsed)
		return nil
	}
	nanos, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid duration %q: use a Go duration such as 30s or 10m", value)
	}
	*d = Duration(nanos)
	return nil
}

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
	AppName                string             `toml:"app_name" json:"appName"`
	ListenAddress          string             `toml:"listen" json:"listenAddress"`
	PublicURL              string             `toml:"public_url" json:"publicUrl"`
	StateDir               string             `toml:"state_dir" json:"stateDir"`
	ArtifactDir            string             `toml:"artifact_dir" json:"artifactDir"`
	RuntimeRoot            string             `toml:"runtime_root" json:"runtimeRoot"`
	RuntimeDownloadTimeout Duration           `toml:"runtime_download_timeout" json:"runtimeDownloadTimeout"`
	OperationRetention     Duration           `toml:"operation_retention" json:"operationRetention"`
	WorkerTimeout          Duration           `toml:"worker_timeout" json:"workerTimeout"`
	OperationTimeout       Duration           `toml:"operation_timeout" json:"operationTimeout"`
	FirecrackerAPITimeout  Duration           `toml:"firecracker_api_timeout" json:"firecrackerApiTimeout"`
	OperationWorkers       int                `toml:"operation_workers" json:"operationWorkers"`
	GuestAgentPort         uint32             `toml:"guest_agent_port" json:"guestAgentPort"`
	GuestAgentCID          uint32             `toml:"guest_agent_cid" json:"guestAgentCID"`
	NetworkCIDR            string             `toml:"network_cidr" json:"networkCIDR"`
	Admin                  AdminConfig        `toml:"admin" json:"admin"`
	Notifications          NotificationConfig `toml:"notifications" json:"notifications"`
}

func Default() Config {
	home, err := os.UserConfigDir()
	if err != nil || home == "" {
		home = ".firecracker-studio"
	} else {
		home = filepath.Join(home, "FirecrackerStudio")
	}
	return Config{
		AppName:                "firecracker-studio",
		ListenAddress:          "127.0.0.1:7822",
		PublicURL:              "http://127.0.0.1:7822",
		StateDir:               home,
		ArtifactDir:            filepath.Join(home, "artifacts"),
		RuntimeRoot:            filepath.Join(home, "runtime"),
		RuntimeDownloadTimeout: Duration(10 * time.Minute),
		OperationRetention:     Duration(24 * time.Hour),
		WorkerTimeout:          Duration(30 * time.Second),
		OperationTimeout:       Duration(30 * time.Minute),
		FirecrackerAPITimeout:  Duration(30 * time.Second),
		OperationWorkers:       2,
		GuestAgentPort:         5000,
		GuestAgentCID:          3,
		NetworkCIDR:            "172.16.0.0/16",
		Admin:                  AdminConfig{Username: "admin"},
		Notifications:          NotificationConfig{SMTPPort: 587},
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
	if c.StateDir == "" || c.ArtifactDir == "" || c.RuntimeRoot == "" {
		return fmt.Errorf("state, artifact, and runtime directories are required")
	}
	if c.RuntimeDownloadTimeout <= 0 {
		return fmt.Errorf("runtime download timeout must be positive")
	}
	if c.OperationRetention <= 0 || c.WorkerTimeout <= 0 || c.OperationTimeout <= 0 || c.FirecrackerAPITimeout <= 0 {
		return fmt.Errorf("retention and all timeouts must be positive")
	}
	if c.GuestAgentPort < 1 || c.GuestAgentPort > 65535 || c.GuestAgentCID < 3 {
		return fmt.Errorf("guest agent port or CID is invalid")
	}
	if _, _, err := net.ParseCIDR(c.NetworkCIDR); err != nil {
		return fmt.Errorf("network CIDR is invalid: %w", err)
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
