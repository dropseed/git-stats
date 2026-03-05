package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultFilename = "git-stats.yml"

type StatConfig struct {
	Run     string `yaml:"run"`
	Type    string `yaml:"type"`
	Default any    `yaml:"default"`
	Goal    string `yaml:"goal"` // "increase" or "decrease"
}

type Config struct {
	Stats map[string]*StatConfig `yaml:"stats"`
	// Preserve key order
	keys []string
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node for config")
	}

	for i := 0; i < len(value.Content)-1; i += 2 {
		key := value.Content[i].Value
		if key == "stats" {
			statsNode := value.Content[i+1]
			if statsNode.Kind != yaml.MappingNode {
				return fmt.Errorf("expected mapping node for stats")
			}

			c.Stats = make(map[string]*StatConfig)
			for j := 0; j < len(statsNode.Content)-1; j += 2 {
				statKey := statsNode.Content[j].Value
				c.keys = append(c.keys, statKey)

				var sc StatConfig
				if err := statsNode.Content[j+1].Decode(&sc); err != nil {
					return err
				}
				c.Stats[statKey] = &sc
			}
		}
	}

	return nil
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultFilename
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) StatsKeys() []string {
	return c.keys
}

func (c *Config) CommandForStat(key string) (string, error) {
	sc, ok := c.Stats[key]
	if !ok {
		return "", fmt.Errorf("unknown stat key: %s", key)
	}
	return sc.Run, nil
}

func (c *Config) TypeForStat(key string) string {
	sc, ok := c.Stats[key]
	if !ok || sc.Type == "" {
		return "number"
	}
	return sc.Type
}

func (c *Config) DefaultForStat(key string) any {
	sc, ok := c.Stats[key]
	if !ok || sc.Default == nil {
		return 0
	}
	return sc.Default
}

func (c *Config) GoalForStat(key string) string {
	sc, ok := c.Stats[key]
	if !ok {
		return ""
	}
	return sc.Goal
}

func ResolveKeys(keys []string, cfg *Config) []string {
	if len(keys) == 0 {
		return cfg.StatsKeys()
	}
	return keys
}
