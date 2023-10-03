package structs

import (
	"bufio"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/gobeam/stringy"
	"github.com/tkcrm/pgxgen/utils"
	"golang.org/x/exp/slices"
)

type Types map[string]TypesParameters
type Structs map[string]*StructParameters

func (s Structs) AddStruct(name string, params *StructParameters) {
	s[name] = params
}

func GetStructs(file_models_str string) Structs {
	r := bufio.NewReader(strings.NewReader(file_models_str))

	structs := make(Structs)

	fileImports := utils.GetGoImportsFromFile(file_models_str)
	currentStruct := &StructParameters{
		Imports: fileImports,
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
			currentStruct = &StructParameters{
				Imports: fileImports,
			}
			continue
		}

		reStartStruct := regexp.MustCompile(`type (\w+) struct {`)
		matches := reStartStruct.FindAllStringSubmatch(line, -1)
		if len(matches) == 1 {
			currentStruct = &StructParameters{
				Name:    matches[0][1],
				Fields:  make([]*StructField, 0),
				Imports: fileImports,
			}
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

func GetMissedStructs(s Structs, scalarTypes Types) []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	res := []string{}

	for _, st := range s {
		for _, stName := range st.GetNestedStructs(scalarTypes) {
			if !slices.Contains(keys, stName) && !slices.Contains(res, stName) {
				res = append(res, stName)
			}
		}
	}

	return res
}

func (s StructParameters) GetNestedStructs(scalarTypes Types) []string {
	res := []string{}

	for _, field := range s.Fields {
		tp := field.Type
		if tp[:1] == "*" {
			tp = tp[1:]
		}

		// skip lowercase type
		if tp != stringy.New(tp).UcFirst() {
			continue
		}

		_, existScalarType := scalarTypes[field.Type]
		if !slices.Contains(res, field.Type) && !existScalarType {
			res = append(res, field.Type)
		}
	}

	return res
}

func FillMissedTypes(allStructs Structs, modelsStructs Structs, scalarTypes Types) {
	missedStructs := GetMissedStructs(modelsStructs, scalarTypes)
	if len(missedStructs) == 0 {
		return
	}

	for _, st := range missedStructs {
		v, ok := allStructs[st]
		if !ok {
			_, ok := scalarTypes[st]
			if ok {
				continue
			}
			log.Fatalf("cannont find struct \"%s\"", st)
		}

		modelsStructs.AddStruct(st, v)
	}

	missedStructs = GetMissedStructs(modelsStructs, scalarTypes)
	if len(missedStructs) == 0 {
		return
	} else {
		FillMissedTypes(allStructs, modelsStructs, scalarTypes)
	}
}
