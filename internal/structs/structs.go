package structs

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/tkcrm/pgxgen/utils"
)

type StructParameters struct {
	Name    string
	Imports []string
	Fields  []*StructField
}

func (s *StructParameters) ExistFieldIndex(name string) int {
	for index, f := range s.Fields {
		if f.Name == name {
			return index
		}
	}
	return -1
}

type StructField struct {
	Name string
	Type string
	Tags map[string]string
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

type TypesParameters struct {
	Name string
	Type string
}

type Types map[string]TypesParameters
type Structs map[string]*StructParameters
type StructSlice []*StructParameters

func (st *StructSlice) ExistStructIndex(name string) (int, *StructParameters) {
	for index, s := range *st {
		if s.Name == name {
			return index, s
		}
	}
	return -1, nil
}

func (st *StructSlice) Sort(priorityNames ...string) error {

	names := make([]string, 0, len(*st))
	for _, name := range priorityNames {
		existStructIndex, _ := st.ExistStructIndex(name)
		if existStructIndex == -1 {
			return fmt.Errorf("sort error: undefined struct %s", name)
		}

		names = append(names, name)
	}

	notPriorityNames := make([]string, 0, len(*st)-len(priorityNames))
	for _, v := range *st {
		if utils.ExistInArray(names, v.Name) {
			continue
		}
		notPriorityNames = append(notPriorityNames, v.Name)
	}
	sort.Strings(notPriorityNames)

	names = append(names, notPriorityNames...)

	sorted := make(StructSlice, 0, len(names))
	for _, n := range names {
		for _, s := range *st {
			if s.Name == n {
				sorted = append(sorted, s)
			}
		}
	}

	*st = sorted

	return nil
}

func ConvertStructsToSlice(st Structs) StructSlice {
	res := make(StructSlice, 0, len(st))

	for _, s := range st {
		res = append(res, s)
	}

	return res
}

func getImports(file_models_str string) []string {
	res := ""
	r := regexp.MustCompile(`(?s)import \(?([^)]+)\)?`)
	match := r.FindAllStringSubmatch(file_models_str, -1)
	if len(match) == 1 {
		res = match[0][1]
	}
	return strings.Split(strings.ReplaceAll(res, "\r\n", "\n"), "\n")
}

func GetStructs(file_models_str string) Structs {
	r := bufio.NewReader(strings.NewReader(file_models_str))

	structs := make(Structs)

	currentStruct := &StructParameters{
		Imports: getImports(file_models_str),
	}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		reCloseStruct := regexp.MustCompile(`^\}`)
		if reCloseStruct.MatchString(line) {
			structs[currentStruct.Name] = currentStruct
			currentStruct = &StructParameters{}
			continue
		}

		reStartStruct := regexp.MustCompile(`type (\w+) struct {`)
		matches := reStartStruct.FindAllStringSubmatch(line, -1)
		if len(matches) == 1 {
			currentStruct = &StructParameters{Name: matches[0][1], Fields: make([]*StructField, 0)}
			continue
		}

		// parse fields
		if currentStruct.Name != "" {
			reField := regexp.MustCompile(`\s*(?P<name>[\w\.]+)\s+(?P<type>[\w\*\.\[\]]+)\s+(?P<tags>\x60.+\x60)?`)
			match := reField.FindStringSubmatch(line)
			if len(match) == 0 {
				continue
			}
			field := StructField{Tags: make(map[string]string)}
			for index, name := range reField.SubexpNames() {
				if index != 0 && name != "" {
					switch name {
					case "name":
						field.Name = match[index]
					case "type":
						field.Type = match[index]
					case "tags":
						reTags := regexp.MustCompile(`(\w+):\"(\w+)\"`)
						match := reTags.FindAllStringSubmatch(match[index], -1)
						for _, m := range match {
							field.Tags[m[1]] = m[2]
						}
					}
				}
			}
			if field.Name != "" {
				currentStruct.Fields = append(currentStruct.Fields, &field)
			}
		}
	}

	return structs
}
