package risk

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadMappingConfigFromFile(path string) (MappingConfig, error) {
	if path == "" {
		return MappingConfig{}, fmt.Errorf("mapping config path is required (--mapping-config)")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return MappingConfig{}, err
	}

	var cfg MappingConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return MappingConfig{}, fmt.Errorf("parse mapping config: %w", err)
	}

	// Defaults
	if cfg.PerSignalCap <= 0 {
		cfg.PerSignalCap = 1.0
	}
	if cfg.Counters == nil {
		cfg.Counters = map[string]CounterRule{}
	}
	if cfg.Quantiles == nil {
		cfg.Quantiles = map[string]QuantileRule{}
	}

	return cfg, nil
}
