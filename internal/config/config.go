package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tkcrm/pgxgen/pkg/logger"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ConfigPaths Flags
	Sqlc        Sqlc
	Pgxgen      Pgxgen
	loadErrs    []error
}

type Flags struct {
	SqlcConfigFilePath   string
	PgxgenConfigFilePath string
}

// LoadConfig return common config with sqlc and pgxgen data
func LoadConfig(cf Flags, version string) (Config, error) {
	cfg := Config{
		ConfigPaths: cf,
		loadErrs:    make([]error, 0),
	}

	// load sqlc config
	var sqlcConfig Sqlc
	sqlcConfigFile, err := os.ReadFile(cf.SqlcConfigFilePath)
	if err != nil {
		cfg.loadErrs = append(cfg.loadErrs, fmt.Errorf("failed to read sqlc config file: %w", err))
	} else {
		// unmashal yaml to sqlc config struct
		if err := yaml.Unmarshal(sqlcConfigFile, &sqlcConfig); err != nil {
			return cfg, err
		}

		// validate sqlc config
		for _, p := range sqlcConfig.Packages {
			if p.Path == "" {
				return cfg, fmt.Errorf("undefined path in sqlc.yaml")
			}
			if p.Queries == "" {
				return cfg, fmt.Errorf("undefined queries in sqlc.yaml")
			}
		}
	}

	// load pgxgen config
	var pgxgenConfig Pgxgen
	pgxgenConfigFile, err := os.ReadFile(cf.PgxgenConfigFilePath)
	if err != nil {
		cfg.loadErrs = append(cfg.loadErrs, fmt.Errorf("failed to read pgxgen config file: %w", err))
	} else {
		// unmashal yaml to pgxgen config struct
		if err := yaml.Unmarshal(pgxgenConfigFile, &pgxgenConfig); err != nil {
			return cfg, err
		}
	}

	cfg.Sqlc = sqlcConfig
	cfg.Pgxgen = pgxgenConfig
	cfg.Pgxgen.Version = version

	return cfg, nil
}

// LoadTestConfig return common config with sqlc and pgxgen data for tests
//
//	configsPath - path where exists `pgxgen.yaml` and `sqlc.yaml` files
func LoadTestConfig(configsPath string) (Config, error) {
	var cfg Config

	// load sqlc config
	var sqlcConfig Sqlc
	sqlcConfigFile, err := os.ReadFile(filepath.Join(configsPath, "sqlc.yaml"))
	if err != nil {
		return cfg, fmt.Errorf("failed to read sqlc config file: %w", err)
	}

	if err := yaml.Unmarshal(sqlcConfigFile, &sqlcConfig); err != nil {
		return cfg, err
	}

	// validate sqlc config
	for _, p := range sqlcConfig.Packages {
		if p.Path == "" {
			return cfg, fmt.Errorf("undefined path in sqlc.yaml")
		}
		if p.Queries == "" {
			return cfg, fmt.Errorf("undefined queries in sqlc.yaml")
		}
	}

	// load pgxgen config
	var pgxgenConfig Pgxgen
	pgxgenConfigFile, err := os.ReadFile(filepath.Join(configsPath, "pgxgen.yaml"))
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(pgxgenConfigFile, &pgxgenConfig); err != nil {
		return cfg, err
	}

	cfg.Sqlc = sqlcConfig
	cfg.Pgxgen = pgxgenConfig
	cfg.Pgxgen.Version = "test-version"

	return cfg, nil
}

// CheckErrors check load errors, print them end exit
func (c *Config) CheckErrors(l logger.Logger) {
	if len(c.loadErrs) == 0 {
		return
	}

	for _, err := range c.loadErrs {
		l.Info(err)
	}
	os.Exit(1)
}
