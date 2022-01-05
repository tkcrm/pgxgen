package gen

import (
	"os"
	"strings"
	"testing"
)

func TestVarPatterns(t *testing.T) {
	type envPair struct {
		key   string
		value string
	}
	type testCase struct {
		env      []envPair
		patterns []string
		expected bool
	}

	cases := []testCase{
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "blah",
				},
			},
			patterns: []string{"pgxgen_FOO"},
			expected: false,
		},
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "blah",
				},
			},
			patterns: []string{"pgxgen_BAR"},
			expected: true,
		},
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "blah",
				},
			},
			patterns: []string{"pgxgen_FOO", "pgxgen_BAR"},
			expected: true,
		},
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "blah",
				},
			},
			patterns: []string{"pgxgen_FOO", "pgxgen_BAR=bim"},
			expected: false,
		},
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "blah",
				},
			},
			patterns: []string{"pgxgen_FOO=", "pgxgen_BAR=blah"},
			expected: true,
		},
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "",
				},
			},
			patterns: []string{"pgxgen_BAR"},
			expected: true,
		},
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "",
				},
			},
			patterns: []string{"pgxgen_BAR="},
			expected: true,
		},
		{
			env: []envPair{
				{
					key:   "pgxgen_BAR",
					value: "blah",
				},
			},
			patterns: []string{"pgxgen_BAR="},
			expected: false,
		},
	}

	prevEnv := os.Environ()
	defer func() {
		for _, setting := range prevEnv {
			eqIdx := strings.Index(setting, "=")
			os.Setenv(setting[:eqIdx], setting[eqIdx+1:])
		}
	}()

	for i, c := range cases {
		os.Clearenv()
		for _, setting := range c.env {
			os.Setenv(setting.key, setting.value)
		}

		actual := anyVarPatternMatches(c.patterns)
		if actual != c.expected {
			t.Fatalf("%d: expected %t, actual %t\n", i, c.expected, actual)
		}
	}
}
