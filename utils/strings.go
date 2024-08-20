package utils

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ToPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	})

	if len(words) == 0 {
		return ""
	}

	titleCaser := cases.Title(language.English)

	var pascalCase string
	for _, word := range words {
		pascalCase += titleCaser.String(strings.ToLower(word))
	}

	return strings.ReplaceAll(pascalCase, "Id", "ID")
}
