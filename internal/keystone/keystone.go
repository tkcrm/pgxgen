package keystone

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tkcrm/modules/pkg/templates"
	"github.com/tkcrm/pgxgen/internal/assets"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/tkcrm/pgxgen/utils"
)

type keystone struct {
	logger logger.Logger
	config config.Config
}

func New(logger logger.Logger, cfg config.Config) generator.IGenerator {
	return &keystone{
		logger: logger,
		config: cfg,
	}
}

func (s *keystone) Generate(_ context.Context, args []string) error {
	s.logger.Infof("generate keystone models")
	timeStart := time.Now()

	if err := s.generateKeystone(); err != nil {
		return err
	}

	s.logger.Infof("keystone models successfully generated in: %s", time.Since(timeStart))

	return nil
}

func (s *keystone) generateKeystone() error {
	for _, params := range s.config.Pgxgen.GenKeystoneFromStruct {
		// validate params
		if err := params.Validate(); err != nil {
			return err
		}

		// get structs from go file
		modelStructs := structs.GetStructsByFilePath(params.InputFilePath)
		modelStructs.RemoveUnexportedFields()

		// get all types from ModelsOutputDir
		scalarTypes := make(structs.Types)

		for _, modelName := range params.SkipModels {
			delete(modelStructs, modelName)
		}

		if err := compileMobxKeystoneModels(s.config.Pgxgen.Version, params, modelStructs, scalarTypes); err != nil {
			return err
		}
	}

	return nil
}

func compileMobxKeystoneModels(ver string, cfg config.GenKeystoneFromStruct, st structs.Structs, sct structs.Types) error {
	if cfg.OutputDir == "" {
		return fmt.Errorf("compile mobx keystone error: undefined output dir")
	}

	if cfg.OutputFileName == "" {
		cfg.OutputFileName = "models.ts"
	}

	mobxKeystoneImports := []string{
		"Model", "tProp", "types", "model", "draft", "modelAction",
		"prop", "clone", "Draft",
	}

	structs := structs.ConvertStructsToSlice(st)
	if cfg.Sort != "" {
		if err := structs.Sort(strings.Split(cfg.Sort, ",")...); err != nil {
			return err
		}
	}

	tctx := tmplKeystoneCtx{
		Structs: structs,
		Imports: map[string][]string{
			"mobx-keystone": mobxKeystoneImports,
			"mobx":          {"computed", "observable"},
		},
		ImportTypes: map[string][]string{
			"@tkcrm/ui": {"FormInstance"},
		},
		DecoratorModelNamePrefix: cfg.DecoratorModelNamePrefix,
		ExportModelSuffix:        cfg.ExportModelSuffix,
		WithSetter:               true,
		Version:                  ver,
	}

	tpl := templates.New()
	tpl.AddFunc("getType", func(t string) string {
		typeWrap, tp := getKeystoneType(cfg, st, sct, t)

		if cfg.WithSetter {
			typeWrap += ".withSetter()"
		}

		return fmt.Sprintf(typeWrap, tp)
	})
	tpl.AddFunc("existFieldId", func(structName, fieldName string) bool {
		for _, s := range st {
			if s.Name != structName {
				continue
			}
			for _, f := range s.Fields {
				if strings.ToLower(f.Name) == fieldName {
					return true
				}
			}
		}

		return false
	})
	tpl.AddFunc("replaceId", func(str string) string {
		return strings.ReplaceAll(str, "ID", "Id")
	})

	compiledRes, err := tpl.Compile(templates.CompileParams{
		TemplateName: "keystoneModelsFile",
		TemplateType: templates.TextTemplateType,
		FS:           assets.TemplatesFS,
		FSPaths: []string{
			"templates/keystone-model.go.tmpl",
		},
		Data: tctx,
	})
	if err != nil {
		return fmt.Errorf("tpl.Compile error: %w", err)
	}

	if err := utils.SaveFile(cfg.OutputDir, cfg.OutputFileName, compiledRes); err != nil {
		return fmt.Errorf("SaveFile error: %w", err)
	}

	if cfg.PrettierCode {
		fmt.Println("prettier generated models ...")
		cmd := exec.Command("npx", "prettier", "--write", filepath.Join(cfg.OutputDir, cfg.OutputFileName))
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

func getKeystoneType(cfg config.GenKeystoneFromStruct, st structs.Structs, sct structs.Types, t string) (typeWrap string, tp string) {
	typeWrap = "tProp(%s)"
	typeWrapUnchecked := "prop<%s>"

	isNullable := strings.Contains(t, "Null") || strings.HasPrefix(t, "*")
	if isNullable {
		t = strings.ReplaceAll(t, "*", "")
		typeWrap = "tProp(types.maybe(%s))"
		typeWrapUnchecked = "prop<%s | undefined>"
	}

	tp = ""
	switch t {
	case "int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32", "pgtype.Int2":
		tp = "types.integer"
		if !isNullable {
			tp += ",0"
		}
	// case "int64", "uint64":
	// 	tp = "types.integer"
	// 	if !isNullable {
	// 		tp += ",0"
	// 	}
	case "int64", "uint64":
		tp = "bigint"
		if !isNullable {
			typeWrap = typeWrapUnchecked + "(0n)"
		} else {
			typeWrap = typeWrapUnchecked + "()"
		}
	case "float32", "float64":
		tp = "types.number"
		if !isNullable {
			tp += ",0"
		}
	case "string", "pgtype.Text":
		tp = "types.string"
		if !isNullable {
			tp += ",\"\""
		}
	case "bool":
		tp = "types.boolean"
		if !isNullable {
			tp += ",false"
		}
	case "bun.NullTime", "time.Time", "pgtype.Time", "pgtype.Timestamp", "timestamppb.Timestamp":
		tp = "types.dateString"
		if !isNullable {
			tp += ",\"\""
		}
	case "uuid.UUID", "uuid.NullUUID":
		tp = "types.string"
		if !isNullable {
			tp += ",\"\""
		}
	case "map[string]interface{}", "map[string]any", "pgtype.JSONB":
		tp = "Record<string, any>"
		typeWrap = typeWrapUnchecked + "({})"
	case "[]byte":
		tp = "Record<string, any>"
		typeWrap = typeWrapUnchecked + "()"
	default:
		_, okst := st[t]
		existScalarItem, oksct := sct[t]
		if okst {
			// var defaultValue string
			// if !isNullable {
			// 	defaultValue = fmt.Sprintf(", new %s%s({})", t, c.ExternalModels.Keystone.ExportModelSuffix)
			// }
			// tp = fmt.Sprintf("types.model(%s%s)%s", t, c.ExternalModels.Keystone.ExportModelSuffix, defaultValue)
			tp = fmt.Sprintf("types.model(%s%s)", t, cfg.ExportModelSuffix)
		} else if oksct {
			typeWrap, tp = getKeystoneType(cfg, st, sct, existScalarItem.Type)
		} else {
			tp = "types.unchecked()"
			fmt.Println("undefined type:", t)
		}
	}

	return typeWrap, tp
}
