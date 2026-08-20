package sources

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type DeploymentYAML struct {
	Name        string            `yaml:"name"`
	Image       string            `yaml:"image"`
	Repository  string            `yaml:"repository"`
	Dockerfile  string            `yaml:"dockerfile"`
	Command     []string          `yaml:"command"`
	Environment map[string]string `yaml:"environment"`
	Ports       []string          `yaml:"ports"`
	Storage     string            `yaml:"storage"`
}

func ParseDeploymentYAML(data []byte) (DeploymentYAML, error) {
	var spec DeploymentYAML
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return DeploymentYAML{}, fmt.Errorf("parse deployment YAML: %w", err)
	}
	if strings.TrimSpace(spec.Image) == "" && strings.TrimSpace(spec.Repository) == "" {
		return DeploymentYAML{}, fmt.Errorf("deployment YAML requires image or repository")
	}
	if spec.Storage == "" {
		spec.Storage = "ephemeral"
	}
	if spec.Storage != "ephemeral" && spec.Storage != "persistent" {
		return DeploymentYAML{}, fmt.Errorf("storage must be ephemeral or persistent")
	}
	for _, port := range spec.Ports {
		if !strings.Contains(port, ":") {
			return DeploymentYAML{}, fmt.Errorf("port mapping %q must use host:guest format", port)
		}
	}
	return spec, nil
}
