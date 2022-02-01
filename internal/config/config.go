package config

type SqlcConfig struct {
	Version  int        `yaml:"version"`
	Packages []Packages `yaml:"packages"`
}

type Packages struct {
	Path    string `yaml:"path"`
	Name    string `yaml:"name"`
	Engine  string `yaml:"engine"`
	Schema  string `yaml:"schema"`
	Queries string `yaml:"queries"`
}
