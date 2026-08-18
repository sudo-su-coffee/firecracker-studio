package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type App struct {
	ctx     context.Context
	baseURL string
	http    *http.Client
	bearer  string
}

func NewApp() *App {
	return &App{baseURL: "http://127.0.0.1:7822", http: &http.Client{Timeout: 30 * time.Second}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
