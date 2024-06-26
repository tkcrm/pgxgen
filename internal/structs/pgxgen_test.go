package structs_test

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/utils"
)

type structParameters struct {
	name   string
	fields []structField
}

type structField struct {
	name  string
	ftype string
	tags  map[string]string
}

var someStructString = "// Code generated with pgxgen. DO NOT EDIT IT\n\n" +
	"type SomeStruct struct {\n" +
	"\tID        int64      `db:\"id\" json:\"id\"`\n" +
	"\tFirstName        string\n" +
	"    CreatedAt time.Time  `db:\"created_at\" json:\"created_at\"`\n" +
	"	UpdatedAt *time.Time `db:\"updated_at\" json:\"updated_at\"`\n" +
	"}\n"

func Test_ParseStruct(t *testing.T) {
	r := bufio.NewReader(strings.NewReader(someStructString))

	structs := make(map[string]structParameters)

	currentStruct := structParameters{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}

		reCloseStruct := regexp.MustCompile(`^\}`)
		if reCloseStruct.MatchString(line) {
			structs[currentStruct.name] = currentStruct
			currentStruct = structParameters{}
			continue
		}

		reStartStruct := regexp.MustCompile(`type (\w+) struct {`)
		matches := reStartStruct.FindAllStringSubmatch(line, -1)
		if len(matches) == 1 {
			currentStruct = structParameters{name: matches[0][1], fields: make([]structField, 0)}
			continue
		}

		// parse fields
		if currentStruct.name != "" {
			reField := regexp.MustCompile(`\s*(?P<name>[\w\.]+)\s+(?P<type>[\w\*\.]+)\s+(?P<tags>\x60.+\x60)?`)
			match := reField.FindStringSubmatch(line)
			if len(match) == 0 {
				continue
			}
			field := structField{tags: make(map[string]string)}
			for index, name := range reField.SubexpNames() {
				if index != 0 && name != "" {
					switch name {
					case "name":
						field.name = match[index]
					case "type":
						field.ftype = match[index]
					case "tags":
						reTags := regexp.MustCompile(`(\w+):\"(\w+)\"`)
						match := reTags.FindAllStringSubmatch(match[index], -1)
						for _, m := range match {
							field.tags[m[1]] = m[2]
						}
					}
				}
			}
			if field.name != "" {
				currentStruct.fields = append(currentStruct.fields, field)
			}
		}
	}

	for _, s := range structs {
		fmt.Printf("%+v\n", s)
	}
}

func Test_UpdateStruct(t *testing.T) {
	r := bufio.NewReader(strings.NewReader(someStructString))
	w := new(strings.Builder)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}

		reCloseStruct := regexp.MustCompile(`^\}`)
		if reCloseStruct.MatchString(line) {
			w.WriteString(line)
			fmt.Println(w.String())
			w.Reset()
			continue
		}

		if w.Len() > 0 {
			w.WriteString(line)
			continue
		}

		reStartStruct := regexp.MustCompile(`type \w+ struct {`)
		if !reStartStruct.MatchString(line) {
			continue
		}
		w.WriteString(line)
		w.WriteString("bun.BaseModel `bun:\"table:users,alias:u\"`\n\n")
	}
}

func TestGetStructsOld(t *testing.T) {
	data, err := utils.ReadFile("../../testdata/teststructs/teststructs.go")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	res := structs.GetStructsOld(string(data))
	spew.Dump(res)
	fmt.Println(time.Since(now))
}

func TestGetStructs(t *testing.T) {
	now := time.Now()
	res := structs.GetStructsByFilePath("../../testdata/teststructs/teststructs.go")
	spew.Dump(res)
	fmt.Println(time.Since(now))
}

func TestGetStructsRemoveUnexported(t *testing.T) {
	now := time.Now()
	res := structs.GetStructsByFilePath("../../testdata/teststructs/unexported.go")
	res.RemoveUnexportedFields()
	spew.Dump(res)
	fmt.Println(time.Since(now))
}
