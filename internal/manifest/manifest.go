package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

type Registry struct {
	Tools []Tool
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read_only"`
	Input       struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
	} `json:"input"`
	Upstream struct {
		Service string            `json:"service"`
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Query   map[string]string `json:"query"`
	} `json:"upstream"`
}

func LoadDir(dir string) (Registry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Registry{}, fmt.Errorf("read manifest dir: %w", err)
	}

	registry := Registry{Tools: make([]Tool, 0, len(entries))}
	seen := map[string]struct{}{}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		tool, err := loadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return Registry{}, err
		}

		if _, ok := seen[tool.Name]; ok {
			return Registry{}, fmt.Errorf("duplicate tool manifest %q", tool.Name)
		}
		seen[tool.Name] = struct{}{}
		registry.Tools = append(registry.Tools, tool)
	}

	if len(registry.Tools) == 0 {
		return Registry{}, fmt.Errorf("manifest dir %q has no tool manifests", dir)
	}

	slices.SortFunc(registry.Tools, func(left, right Tool) int {
		if left.Name < right.Name {
			return -1
		}
		if left.Name > right.Name {
			return 1
		}
		return 0
	})

	return registry, nil
}

func (r Registry) Tool(name string) (Tool, bool) {
	for _, tool := range r.Tools {
		if tool.Name == name {
			return tool, true
		}
	}
	return Tool{}, false
}

func loadFile(path string) (Tool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Tool{}, fmt.Errorf("read manifest %q: %w", path, err)
	}

	var tool Tool
	if err := json.Unmarshal(payload, &tool); err != nil {
		return Tool{}, fmt.Errorf("decode manifest %q: %w", path, err)
	}

	if err := validate(tool); err != nil {
		return Tool{}, fmt.Errorf("validate manifest %q: %w", path, err)
	}

	return tool, nil
}

func validate(tool Tool) error {
	if tool.Name == "" {
		return fmt.Errorf("name is required")
	}
	if tool.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(tool.Input.Required) == 0 {
		return fmt.Errorf("at least one required input is required")
	}
	if len(tool.Input.Properties) == 0 {
		return fmt.Errorf("input properties are required")
	}
	if tool.Upstream.Service == "" {
		return fmt.Errorf("upstream service is required")
	}
	if tool.Upstream.Method == "" {
		return fmt.Errorf("upstream method is required")
	}
	if tool.Upstream.Path == "" {
		return fmt.Errorf("upstream path is required")
	}
	return nil
}
