package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	contents := "listen = \"0.0.0.0:7822\"\npublic_url = \"https://studio.example.com\"\n[admin]\nusername = \"operator\"\nemail = \"operator@example.com\"\npassword_hash = \"hash\"\n"
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddress != "0.0.0.0:7822" || cfg.Admin.Username != "operator" || cfg.PublicURL != "https://studio.example.com" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestDefaultLoopbackDoesNotRequireCredentials(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
