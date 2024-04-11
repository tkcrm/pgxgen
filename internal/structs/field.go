package structs

import (
	"fmt"
	"sort"

	"github.com/gobeam/stringy"
)

type StructField struct {
	Name string
	Type string
	Tags map[string]string

	exprData *fieldExprData
}

func (s *StructField) GetGoTag() string {
	tag := ""

	keys := make([]string, 0, len(s.Tags))
	for k := range s.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if tag != "" {
			tag += " "
		}
		tag += fmt.Sprintf("%s:\"%s\"", k, s.Tags[k])
	}

	return tag
}

func (s StructField) IsExported() bool {
	return s.Name == stringy.New(s.Name).UcFirst()
}
