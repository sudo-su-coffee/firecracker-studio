package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GitHubSource struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Dockerfile string `json:"dockerfile,omitempty"`
	YAMLPath   string `json:"yamlPath,omitempty"`
	RawURL     string `json:"rawUrl,omitempty"`
}

type repoResponse struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
}

type commitResponse struct {
	SHA string `json:"sha"`
}

func ResolveGitHub(ctx context.Context, reference string) (GitHubSource, error) {
	repo, err := parseRepository(reference)
	if err != nil {
		return GitHubSource{}, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	request := func(path string, out any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
		}
		return json.NewDecoder(resp.Body).Decode(out)
	}
	var repository repoResponse
	if err := request("/repos/"+url.PathEscape(repo), &repository); err != nil {
		return GitHubSource{}, fmt.Errorf("resolve GitHub repository: %w", err)
	}
	var commit commitResponse
	if err := request("/repos/"+url.PathEscape(repo)+"/commits/"+url.PathEscape(repository.DefaultBranch), &commit); err != nil {
		return GitHubSource{}, fmt.Errorf("resolve GitHub commit: %w", err)
	}
	return GitHubSource{Repository: repository.FullName, Commit: commit.SHA, Dockerfile: "Dockerfile", RawURL: "https://raw.githubusercontent.com/" + repository.FullName + "/" + commit.SHA + "/Dockerfile"}, nil
}

func parseRepository(reference string) (string, error) {
	raw := strings.TrimSpace(reference)
	raw = strings.TrimSuffix(raw, ".git")
	if strings.HasPrefix(raw, "git@github.com:") {
		raw = "https://github.com/" + strings.TrimPrefix(raw, "git@github.com:")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host != "github.com" {
		return "", fmt.Errorf("GitHub reference must be a github.com repository URL")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("GitHub repository owner and name are required")
	}
	return parts[0] + "/" + parts[1], nil
}
