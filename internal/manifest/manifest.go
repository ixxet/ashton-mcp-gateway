package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

type runtimeSupport struct {
	required []string
	query    map[string]string
}

var supportedTools = map[string]runtimeSupport{
	"athena.get_current_occupancy": {
		required: []string{"facility_id"},
		query: map[string]string{
			"facility": "facility_id",
		},
	},
	"athena.get_current_zone_occupancy": {
		required: []string{"facility_id", "zone_id"},
		query: map[string]string{
			"facility": "facility_id",
			"zone":     "zone_id",
		},
	},
}

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
	if err := validateCurrentRuntimeSupport(tool); err != nil {
		return err
	}
	return nil
}

func validateCurrentRuntimeSupport(tool Tool) error {
	support, ok := supportedTools[tool.Name]
	if !ok {
		return fmt.Errorf("current runtime supports only the Tracer 15 ATHENA occupancy tools")
	}
	if !tool.ReadOnly {
		return fmt.Errorf("current runtime supports read-only tools only")
	}
	if !slices.Equal(tool.Input.Required, support.required) {
		return fmt.Errorf("current runtime requires exact required inputs %v", support.required)
	}
	for _, required := range support.required {
		if _, ok := tool.Input.Properties[required]; !ok {
			return fmt.Errorf("current runtime requires %s input metadata", required)
		}
	}
	if tool.Upstream.Service != "athena" {
		return fmt.Errorf("current runtime supports ATHENA upstream only")
	}
	if tool.Upstream.Method != "GET" {
		return fmt.Errorf("current runtime supports GET upstream methods only")
	}
	if tool.Upstream.Path != "/api/v1/presence/count" {
		return fmt.Errorf("current runtime supports ATHENA occupancy path only")
	}
	if len(tool.Upstream.Query) != len(support.query) {
		return fmt.Errorf("current runtime requires exact query mapping %v", support.query)
	}
	for key, value := range support.query {
		if tool.Upstream.Query[key] != value {
			return fmt.Errorf("current runtime requires %s query mapped from %s", key, value)
		}
	}

	return nil
}
