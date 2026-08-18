package connections

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Store persists server profiles in the native per-user application config directory.
// Bearer tokens never appear in the public Server JSON because Server.BearerToken is
// intentionally tagged json:"-".
type Store struct {
	path string
}

type persistedServer struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Kind        string     `json:"kind"`
	Username    string     `json:"username,omitempty"`
	Health      string     `json:"health"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
	Active      bool       `json:"active"`
	BearerToken string     `json:"bearerToken,omitempty"`
}

func NewStore() *Store {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	if runtime.GOOS == "windows" {
		return &Store{path: filepath.Join(base, "BlackLoverTech", "Firecracker Studio", "servers.json")}
	}
	return &Store{path: filepath.Join(base, "firecracker-studio", "servers.json")}
}

func (s *Store) Path() string { return s.path }

func (s *Store) Load() ([]Server, error) {
	payload, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []Server{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read server profiles: %w", err)
	}
	var records []persistedServer
	if err := json.Unmarshal(payload, &records); err != nil {
		return nil, fmt.Errorf("decode server profiles: %w", err)
	}
	servers := make([]Server, 0, len(records))
	for _, record := range records {
		servers = append(servers, Server{
			ID: record.ID, Name: record.Name, URL: record.URL, Kind: record.Kind,
			Username: record.Username, Health: record.Health, LastChecked: record.LastChecked,
			Active: record.Active, BearerToken: record.BearerToken,
		})
	}
	return servers, nil
}

func (s *Store) Save(servers []Server) error {
	records := make([]persistedServer, 0, len(servers))
	for _, server := range servers {
		records = append(records, persistedServer{
			ID: server.ID, Name: server.Name, URL: server.URL, Kind: server.Kind,
			Username: server.Username, Health: server.Health, LastChecked: server.LastChecked,
			Active: server.Active, BearerToken: server.BearerToken,
		})
	}
	payload, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("encode server profiles: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create server profile directory: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(payload, '\n'), 0600); err != nil {
		return fmt.Errorf("write server profiles: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("commit server profiles: %w", err)
	}
	return nil
}
