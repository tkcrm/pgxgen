package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// versionDetect is used to detect the config version before full parsing.
type versionDetect struct {
	Version string `yaml:"version"`
}

// LoadV2Config loads and validates a v2 pgxgen config file.
func LoadV2Config(path string) (*V2Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Detect version
	var vd versionDetect
	if err := yaml.Unmarshal(data, &vd); err != nil {
		return nil, fmt.Errorf("failed to parse config version: %w", err)
	}

	switch vd.Version {
	case "2":
		// proceed with v2 parsing
	case "1", "":
		return nil, fmt.Errorf(
			"config version %q detected. Run 'pgxgen migrate' to upgrade to v2",
			vd.Version,
		)
	default:
		return nil, fmt.Errorf("unsupported config version: %s", vd.Version)
	}

	var cfg V2Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse v2 config: %w", err)
	}

	// Apply defaults
	for i := range cfg.Schemas {
		applySchemaDefaults(&cfg.Schemas[i])
	}

	return &cfg, nil
}

// applySchemaDefaults merges default method configs into each table.
func applySchemaDefaults(schema *SchemaConfig) {
	if schema.Defaults == nil || schema.Defaults.Crud == nil {
		return
	}

	defaultMethods := schema.Defaults.Crud.Methods
	if defaultMethods == nil {
		return
	}

	for tableName, table := range schema.Tables {
		if table.Crud == nil {
			continue
		}
		if table.Crud.Methods == nil {
			continue
		}

		for methodName, methodCfg := range table.Crud.Methods {
			if methodCfg == nil {
				// Method declared with no config (e.g., `get: {}`) — apply defaults
				if defaultMethod, ok := defaultMethods[methodName]; ok {
					merged := mergeMethod(defaultMethod, nil)
					table.Crud.Methods[methodName] = merged
				}
				continue
			}

			// Merge default into per-table method
			if defaultMethod, ok := defaultMethods[methodName]; ok {
				merged := mergeMethod(defaultMethod, methodCfg)
				table.Crud.Methods[methodName] = merged
			}
		}

		schema.Tables[tableName] = table
	}
}

// mergeMethod merges a default method config with a per-table override.
// Per-table values take precedence over defaults.
func mergeMethod(base, override *MethodConfig) *MethodConfig {
	if base == nil {
		return override
	}
	if override == nil {
		// Deep copy base
		result := *base
		return &result
	}

	result := *base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Returning != "" {
		result.Returning = override.Returning
	}
	if override.SkipColumns != nil {
		result.SkipColumns = override.SkipColumns
	}
	if override.ColumnValues != nil {
		result.ColumnValues = override.ColumnValues
	}
	if override.Where != nil {
		result.Where = override.Where
	}
	if override.WhereAdditional != nil {
		result.WhereAdditional = override.WhereAdditional
	}
	if override.Limit {
		result.Limit = override.Limit
	}
	if override.Order != nil {
		result.Order = override.Order
	}

	return &result
}
