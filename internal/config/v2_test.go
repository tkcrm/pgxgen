package config

import "testing"

func TestResolveOutputDir(t *testing.T) {
	tests := []struct {
		name     string
		schema   SchemaConfig
		table    string
		expected string
	}{
		{
			name: "no prefix/suffix (backward compat)",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					OutputDirPrefix: "repos",
				},
			},
			table:    "users",
			expected: "repos/users",
		},
		{
			name: "with package_prefix",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					OutputDirPrefix: "repos",
					PackagePrefix:   "repo_",
				},
			},
			table:    "users",
			expected: "repos/repo_users",
		},
		{
			name: "with package_suffix",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					OutputDirPrefix: "repos",
					PackageSuffix:   "_repo",
				},
			},
			table:    "users",
			expected: "repos/users_repo",
		},
		{
			name: "with both prefix and suffix",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					OutputDirPrefix: "repos",
					PackagePrefix:   "repo_",
					PackageSuffix:   "_store",
				},
			},
			table:    "users",
			expected: "repos/repo_users_store",
		},
		{
			name: "pattern B single repo unaffected",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					OutputDir:     "single_repo",
					PackagePrefix: "repo_",
				},
			},
			table:    "users",
			expected: "single_repo",
		},
		{
			name:     "nil defaults",
			schema:   SchemaConfig{},
			table:    "users",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.schema.ResolveOutputDir(tt.table)
			if got != tt.expected {
				t.Errorf("ResolveOutputDir(%q) = %q, want %q", tt.table, got, tt.expected)
			}
		})
	}
}

func TestResolveQueriesDir(t *testing.T) {
	tests := []struct {
		name     string
		schema   SchemaConfig
		table    string
		expected string
	}{
		{
			name: "no prefix/suffix",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					QueriesDirPrefix: "queries",
				},
			},
			table:    "users",
			expected: "queries/users",
		},
		{
			name: "package_prefix does not affect queries dir",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					QueriesDirPrefix: "queries",
					PackagePrefix:    "repo_",
				},
			},
			table:    "users",
			expected: "queries/users",
		},
		{
			name: "pattern B single repo unaffected",
			schema: SchemaConfig{
				Defaults: &DefaultsConfig{
					QueriesDir:    "single_queries",
					PackagePrefix: "repo_",
				},
			},
			table:    "users",
			expected: "single_queries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.schema.ResolveQueriesDir(tt.table)
			if got != tt.expected {
				t.Errorf("ResolveQueriesDir(%q) = %q, want %q", tt.table, got, tt.expected)
			}
		})
	}
}
