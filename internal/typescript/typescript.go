package typescript

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/tkcrm/modules/pkg/templates"
	"github.com/tkcrm/pgxgen/internal/assets"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/tkcrm/pgxgen/utils"
)

type typescript struct {
	logger logger.Logger
	config config.Config
}

func New(logger logger.Logger, cfg config.Config) generator.IGenerator {
	return &typescript{
		logger: logger,
		config: cfg,
	}
}

type tmplTypescriptCtx struct {
	Version          string
	Structs          structs.StructSlice
	ExportTypePrefix string
	ExportTypeSuffix string
}

func (s *typescript) Generate(_ context.Context, args []string) error {
	s.logger.Infof("generate typescript code")
	timeStart := time.Now()

	if err := s.generateTypescript(args); err != nil {
		return err
	}

	s.logger.Infof("typescript code successfully generated in: %s", time.Since(timeStart))

	return nil
}

func (s *typescript) generateTypescript(args []string) error {
	for _, config := range s.config.Pgxgen.GenTypescriptFromStructs {
		if config.OutputDir == "" {
			return fmt.Errorf("output dir is empty")
		}

		if config.OutputFileName == "" {
			return fmt.Errorf("output file name is empty")
		}

		dirItems, err := os.ReadDir(config.Path)
		if err != nil {
			return err
		}

		_structs := make(structs.Structs)

		for _, item := range dirItems {
			if item.IsDir() || !strings.HasSuffix(item.Name(), ".go") {
				continue
			}

			for key, value := range structs.GetStructsByFilePath(filepath.Join(config.Path, item.Name())) {
				_structs[key] = value
			}

		}

		// Filter structs
		for key, value := range _structs {
			var deleteStruct *bool
			for _, restring := range config.IncludeStructNamesRegexp {
				r := regexp.MustCompile(restring)
				if !r.MatchString(value.Name) {
					if deleteStruct == nil {
						tr := true
						deleteStruct = &tr
					}
				} else {
					fs := false
					deleteStruct = &fs
				}
			}

			for _, restring := range config.ExcludeStructNamesRegexp {
				r := regexp.MustCompile(restring)
				if r.MatchString(value.Name) {
					tr := true
					deleteStruct = &tr
				}
			}

			if deleteStruct != nil && *deleteStruct {
				delete(_structs, key)
			}
		}

		structsSlice := structs.ConvertStructsToSlice(_structs)
		if err := structsSlice.Sort(); err != nil {
			return err
		}

		if err := s.compileTypescript(config, structsSlice); err != nil {
			return err
		}
	}

	return nil
}

func (s *typescript) compileTypescript(c config.GenTypescriptFromStructs, st structs.StructSlice) error {
	tpl := templates.New()

	// cdata := templates.CompileData{
	// 	OutputDir:      c.OutputDir,
	// 	OutputFileName: c.OutputFileName,
	// }

	// funcs := templates.DefaultTmplFuncs
	tpl.AddFunc("isEmptyFields", func(fields []*structs.StructField) bool {
		return len(fields) == 0
	})
	tpl.AddFunc("filterFields", func(fields []*structs.StructField) []*structs.StructField {
		resFields := make([]*structs.StructField, 0)
		for _, f := range fields {
			if unicode.IsLower(rune(f.Name[0])) {
				continue
			}
			resFields = append(resFields, f)
		}

		return resFields
	})
	tpl.AddFunc("isNullable", func(t string) bool {
		return strings.Contains(t, "*")
	})
	tpl.AddFunc("getType", func(t string) string {
		return getTypescriptType(st, t)
	})

	// tmpl := template.Must(
	// 	template.New("").
	// 		Funcs(funcs).
	// 		ParseFS(
	// 			templates.Templates,
	// 			"templates/typescript.go.tmpl",
	// 		),
	// )

	tctx := tmplTypescriptCtx{
		Version:          s.config.Pgxgen.Version,
		Structs:          st,
		ExportTypePrefix: c.ExportTypePrefix,
		ExportTypeSuffix: c.ExportTypeSuffix,
	}

	// var b bytes.Buffer
	// w := bufio.NewWriter(&b)
	// err := tmpl.ExecuteTemplate(w, "typescriptFile", &tctx)
	// w.Flush()
	// if err != nil {
	// 	return fmt.Errorf("execte template error: %s", err.Error())
	// }

	// cdata.Data = b.Bytes()

	compiledRes, err := tpl.Compile(templates.CompileParams{
		TemplateName: "typescriptFile",
		TemplateType: templates.TextTemplateType,
		FS:           assets.TemplatesFS,
		FSPaths: []string{
			"templates/typescript.go.tmpl",
		},
		Data: tctx,
	})
	if err != nil {
		return fmt.Errorf("tpl.Compile error: %w", err)
	}

	if err := utils.SaveFile(c.OutputDir, c.OutputFileName, compiledRes); err != nil {
		return fmt.Errorf("SaveFile error: %w", err)
	}

	if c.PrettierCode {
		fmt.Println("prettier generated typescript code ...")
		cmd := exec.Command("npx", "prettier", "--write", filepath.Join(c.OutputDir, c.OutputFileName))
		stdout, err := cmd.Output()
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}
		if len(stdout) > 0 {
			fmt.Println(string(stdout))
		}
	}

	return nil
}

func getTypescriptType(st structs.StructSlice, t string) (tp string) {
	t = strings.ReplaceAll(t, "*", "")

	tp = ""
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		tp = "number"
	case "float32", "float64":
		tp = "number"
	case "string":
		tp = "string"
	case "bool":
		tp = "boolean"
	case "bun.NullTime", "time.Time", "timestamppb.Timestamp":
		tp = "Date"
	case "map[string]interface{}", "pgtype.JSONB":
		tp = "Record<string, any>"
	default:
		tp = "any"
		fmt.Println("undefined type:", t)
	}
	return tp
}
