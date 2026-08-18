package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sudo-su-coffee/firecracker-studio/internal/connections"
)

type App struct {
	ctx     context.Context
	baseURL string
	http    *http.Client
	bearer  string
	servers []connections.Server
}

func NewApp() *App {
	return &App{http: &http.Client{Timeout: 30 * time.Second}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Servers() []connections.Server { return append([]connections.Server(nil), a.servers...) }

func (a *App) AddServer(name, baseURL, kind, username, bearer string) (connections.Server, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return connections.Server{}, fmt.Errorf("worker URL is required")
	}
	server := connections.Server{ID: uuid.NewString(), Name: name, URL: baseURL, Kind: kind, Username: username, BearerToken: bearer, Health: "unchecked"}
	a.servers = append(a.servers, server)
	if _, err := a.checkURL(baseURL, bearer); err != nil {
		server.Health = "unhealthy"
	} else {
		server.Health = "healthy"
	}
	now := time.Now().UTC()
	server.LastChecked = &now
	for i := range a.servers {
		if a.servers[i].ID == server.ID {
			a.servers[i] = server
		}
	}
	return server, nil
}

func (a *App) CheckServer(id string) (connections.Server, error) {
	for i := range a.servers {
		if a.servers[i].ID == id {
			status, err := a.checkURL(a.servers[i].URL, a.servers[i].BearerToken)
			now := time.Now().UTC()
			a.servers[i].LastChecked = &now
			if err != nil {
				a.servers[i].Health = "unhealthy"
			} else {
				a.servers[i].Health = status
			}
			return a.servers[i], err
		}
	}
	return connections.Server{}, fmt.Errorf("server %q not found", id)
}

func (a *App) SwitchServer(id string) (connections.Server, error) {
	for i := range a.servers {
		if a.servers[i].ID == id {
			if a.servers[i].Health != "healthy" {
				if _, err := a.CheckServer(id); err != nil {
					return connections.Server{}, fmt.Errorf("server health check failed: %w", err)
				}
			}
			a.baseURL = a.servers[i].URL
			a.bearer = a.servers[i].BearerToken
			for j := range a.servers {
				a.servers[j].Active = j == i
			}
			return a.servers[i], nil
		}
	}
	return connections.Server{}, fmt.Errorf("server %q not found", id)
}

func (a *App) SetConnection(baseURL, bearer string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("worker URL is required")
	}
	a.baseURL = baseURL
	a.bearer = bearer
	return nil
}

func (a *App) Connection() map[string]string {
	return map[string]string{"url": a.baseURL}
}

func (a *App) Health() (map[string]any, error) {
	var out map[string]any
	return out, a.request(http.MethodGet, "/api/v1/health", nil, &out)
}

func (a *App) BaseImages() (map[string]any, error) {
	var out map[string]any
	return out, a.request(http.MethodGet, "/api/v1/base-images", nil, &out)
}

func (a *App) Images() (map[string]any, error) {
	var out map[string]any
	return out, a.request(http.MethodGet, "/api/v1/images", nil, &out)
}

func (a *App) Operations() (map[string]any, error) {
	var out map[string]any
	return out, a.request(http.MethodGet, "/api/v1/operations", nil, &out)
}

func (a *App) VMs() (map[string]any, error) {
	var out map[string]any
	return out, a.request(http.MethodGet, "/api/v1/vms", nil, &out)
}

func (a *App) Convert(source, sourceType, baseProfile string) (map[string]any, error) {
	body := map[string]string{"source": source, "sourceType": sourceType, "baseProfile": baseProfile}
	var out map[string]any
	return out, a.request(http.MethodPost, "/api/v1/conversions", body, &out)
}

func (a *App) CreateVM(artifactDigest string, vcpus, memoryMiB int) (map[string]any, error) {
	body := map[string]any{"artifactDigest": artifactDigest, "vcpus": vcpus, "memoryMiB": memoryMiB}
	var out map[string]any
	return out, a.request(http.MethodPost, "/api/v1/vms", body, &out)
}

func (a *App) VMAction(id, action string) (map[string]any, error) {
	if action != "start" && action != "stop" {
		return nil, fmt.Errorf("unsupported VM action %q", action)
	}
	var out map[string]any
	return out, a.request(http.MethodPost, "/api/v1/vms/"+id+"/"+action, map[string]any{}, &out)
}

func (a *App) Snapshot(id, action, snapshotPath, memoryPath string) (map[string]any, error) {
	if action != "create" && action != "restore" {
		return nil, fmt.Errorf("unsupported snapshot action %q", action)
	}
	var out map[string]any
	path := "/api/v1/vms/" + id + "/snapshots"
	if action == "restore" {
		path += "/restore"
	}
	body := map[string]string{"snapshotPath": snapshotPath, "memoryPath": memoryPath}
	return out, a.request(http.MethodPost, path, body, &out)
}

func (a *App) checkURL(baseURL, bearer string) (string, error) {
	req, err := http.NewRequestWithContext(a.contextOrBackground(), http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/health", nil)
	if err != nil {
		return "", err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("health check returned HTTP %d", resp.StatusCode)
	}
	return "healthy", nil
}

func (a *App) request(method, path string, body any, out any) error {
	if a.http == nil {
		a.http = &http.Client{Timeout: 30 * time.Second}
	}
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(a.contextOrBackground(), method, a.baseURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+a.bearer)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("worker returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (a *App) contextOrBackground() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}
