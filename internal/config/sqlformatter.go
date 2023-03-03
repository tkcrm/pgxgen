package config

type SQLFormatter struct {
	Paths SQLFormatterPaths `yaml:"paths"`
}

type SQLFormatterPaths map[string]SQLFormatterPathsItem

type SQLFormatterPathsItem struct {
	FileNameRegexp string `yaml:"file_name_regexp"`
}
