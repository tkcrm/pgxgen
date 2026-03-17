package models

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jinzhu/inflection"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/sqlparser"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
	"github.com/tkcrm/pgxgen/internal/sqlparser/typemap"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/tkcrm/pgxgen/utils" // for SaveFile
)

// Common Go acronyms that should be fully uppercased.
var acronyms = map[string]string{
	"id":    "ID",
	"ids":   "IDs",
	"url":   "URL",
	"uri":   "URI",
	"api":   "API",
	"http":  "HTTP",
	"https": "HTTPS",
	"ip":    "IP",
	"sql":   "SQL",
	"ssh":   "SSH",
	"tcp":   "TCP",
	"udp":   "UDP",
	"uid":   "UID",
	"uuid":  "UUID",
	"json":  "JSON",
	"xml":   "XML",
	"html":  "HTML",
	"css":   "CSS",
	"js":    "JS",
	"cpu":   "CPU",
	"gpu":   "GPU",
	"os":    "OS",
	"db":    "DB",
	"io":    "IO",
	"eof":   "EOF",
	"tls":   "TLS",
	"ttl":   "TTL",
	"dns":   "DNS",
}

// GenerateFromV2Config generates Go models from SQL schema using v2 config.
// If cat is nil, the schema will be parsed from schemaDir.
func GenerateFromV2Config(
	l logger.Logger,
	configDir string,
	modelsCfg *config.ModelsConfig,
	engine string,
	schemaDir string,
	cat *catalog.Catalog,
	sqlcCfg *config.SqlcConfig,
) error {
	if modelsCfg == nil {
		return nil
	}

	if engine == "" {
		engine = "postgresql"
	}

	// Parse schema if not provided
	if cat == nil {
		resolvedSchemaDir := schemaDir
		if !filepath.IsAbs(resolvedSchemaDir) {
			absDir, err := filepath.Abs(filepath.Join(configDir, resolvedSchemaDir))
			if err != nil {
				return fmt.Errorf("resolve schema dir: %w", err)
			}
			resolvedSchemaDir = absDir
		}

		parser, err := sqlparser.NewParser(engine)
		if err != nil {
			return fmt.Errorf("create parser: %w", err)
		}

		files, err := sqlparser.ResolveSchemaFiles(resolvedSchemaDir)
		if err != nil {
			return fmt.Errorf("resolve schema files: %w", err)
		}

		var parseErr error
		cat, parseErr = parser.ParseSchema(files)
		if parseErr != nil {
			return fmt.Errorf("parse schema: %w", parseErr)
		}
	}

	mapper, err := typemap.NewTypeMapper(engine)
	if err != nil {
		return fmt.Errorf("create type mapper: %w", err)
	}

	sqlPackage := modelsCfg.SqlPackage
	if sqlPackage == "" {
		sqlPackage = "pgx/v5"
	}

	// replace_nullable_types implies emit_pointers_for_null
	emitPointers := modelsCfg.EmitPointersForNull || modelsCfg.ReplaceNullableTypes

	mapOpts := typemap.Options{
		SqlPackage:          sqlPackage,
		EmitPointersForNull: emitPointers,
	}

	var sqlcOverrides *config.SqlcOverridesConfig
	var sqlcDefaults *config.SqlcDefaultsConfig
	if sqlcCfg != nil {
		sqlcOverrides = sqlcCfg.Overrides
		sqlcDefaults = sqlcCfg.Defaults
	}

	code, err := renderModels(modelsCfg, cat, mapper, mapOpts, sqlcOverrides, sqlcDefaults)
	if err != nil {
		return fmt.Errorf("render models: %w", err)
	}

	outputDir := modelsCfg.OutputDir
	if !filepath.IsAbs(outputDir) {
		absDir, err := filepath.Abs(filepath.Join(configDir, outputDir))
		if err != nil {
			return fmt.Errorf("resolve output dir: %w", err)
		}
		outputDir = absDir
	}

	if err := utils.SaveFile(outputDir, modelsCfg.GetOutputFileName(), code); err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	l.Infof("models generated: %s/%s", outputDir, modelsCfg.GetOutputFileName())
	return nil
}

func renderModels(
	cfg *config.ModelsConfig,
	cat *catalog.Catalog,
	mapper typemap.TypeMapper,
	mapOpts typemap.Options,
	sqlcOverrides *config.SqlcOverridesConfig,
	sqlcDefaults *config.SqlcDefaultsConfig,
) ([]byte, error) {
	raw, err := renderModelsRaw(cfg, cat, mapper, mapOpts, sqlcOverrides, sqlcDefaults)
	if err != nil {
		return nil, err
	}

	// format.Source is fast (~1ms), goimports is slow (~2-3s).
	// We resolve imports ourselves in renderModelsRaw, so format.Source is enough.
	result, err := format.Source(raw)
	if err != nil {
		return nil, fmt.Errorf("format code: %w", err)
	}

	return result, nil
}

// renderModelsRaw generates Go source without formatting.
func renderModelsRaw(
	cfg *config.ModelsConfig,
	cat *catalog.Catalog,
	mapper typemap.TypeMapper,
	mapOpts typemap.Options,
	sqlcOverrides *config.SqlcOverridesConfig,
	sqlcDefaults *config.SqlcDefaultsConfig,
) ([]byte, error) {
	// Collect all enums and tables across schemas
	var allEnums []*catalog.Enum
	var allTables []*catalog.Table
	for _, s := range cat.Schemas {
		allEnums = append(allEnums, s.Enums...)
		allTables = append(allTables, s.Tables...)
	}

	sort.Slice(allEnums, func(i, j int) bool {
		return allEnums[i].Name < allEnums[j].Name
	})
	sort.Slice(allTables, func(i, j int) bool {
		return inflection.Singular(allTables[i].Name) < inflection.Singular(allTables[j].Name)
	})

	// First pass: render body to collect which types are used
	var body bytes.Buffer
	for _, enum := range allEnums {
		renderEnum(&body, enum)
	}
	for _, table := range allTables {
		renderStruct(&body, cfg, table, mapper, allEnums, mapOpts, sqlcOverrides, sqlcDefaults)
	}

	// Collect imports from the rendered body
	importsNeeded := collectImports(body.String(), sqlcOverrides)

	// Build final output with imports
	var buf bytes.Buffer
	buf.WriteString("// Code generated by pgxgen. DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", cfg.PackageName))

	if len(importsNeeded) > 0 {
		buf.WriteString("import (\n")
		// Sort imports for deterministic output
		sortedImports := make([]string, 0, len(importsNeeded))
		for imp := range importsNeeded {
			sortedImports = append(sortedImports, imp)
		}
		sort.Strings(sortedImports)
		for _, imp := range sortedImports {
			buf.WriteString(fmt.Sprintf("\t%q\n", imp))
		}
		buf.WriteString(")\n\n")
	}

	buf.Write(body.Bytes())

	return buf.Bytes(), nil
}

// knownTypeImports maps Go type prefixes/names to their import paths.
var knownTypeImports = map[string]string{
	"time.":       "time",
	"sql.":        "database/sql",
	"uuid.":       "github.com/google/uuid",
	"pgtype.":     "github.com/jackc/pgx/v5/pgtype",
	"netip.":      "net/netip",
	"net.":        "net",
	"json.":       "encoding/json",
	"pqtype.":     "github.com/sqlc-dev/pqtype",
}

// collectImports scans rendered Go code for type references and returns needed imports.
func collectImports(code string, sqlcOverrides *config.SqlcOverridesConfig) map[string]struct{} {
	imports := make(map[string]struct{})

	// Check known types
	for prefix, importPath := range knownTypeImports {
		if strings.Contains(code, prefix) {
			imports[importPath] = struct{}{}
		}
	}

	// Check sqlc type overrides with import paths
	if sqlcOverrides != nil {
		for _, to := range sqlcOverrides.Types {
			goType := extractGoType(to.GoType)
			if strings.Contains(code, goType) {
				if imp := extractImportPath(to.GoType); imp != "" {
					imports[imp] = struct{}{}
				}
			}
		}
		for _, co := range sqlcOverrides.Columns {
			if co.GoType != nil {
				goType := extractGoType(co.GoType)
				if strings.Contains(code, goType) {
					if imp := extractImportPath(co.GoType); imp != "" {
						imports[imp] = struct{}{}
					}
				}
			}
		}
	}

	return imports
}

// extractImportPath gets the import path from a go_type value.
func extractImportPath(v interface{}) string {
	switch t := v.(type) {
	case string:
		// "github.com/google/uuid.UUID" → "github.com/google/uuid"
		if idx := strings.LastIndex(t, "."); idx > 0 {
			return t[:idx]
		}
	case map[string]interface{}:
		if imp, ok := t["import"].(string); ok {
			return imp
		}
	}
	return ""
}

func renderEnum(buf *bytes.Buffer, enum *catalog.Enum) {
	typeName := toCamelCase(enum.Name)

	buf.WriteString(fmt.Sprintf("type %s string\n\n", typeName))
	buf.WriteString("const (\n")
	for _, val := range enum.Values {
		constName := typeName + toCamelCase(val)
		buf.WriteString(fmt.Sprintf("\t%s %s = %q\n", constName, typeName, val))
	}
	buf.WriteString(")\n\n")

	buf.WriteString(fmt.Sprintf("func (e %s) Valid() bool {\n", typeName))
	buf.WriteString("\tswitch e {\n")
	buf.WriteString("\tcase ")
	for i, val := range enum.Values {
		if i > 0 {
			buf.WriteString(",\n\t\t")
		}
		buf.WriteString(typeName + toCamelCase(val))
	}
	buf.WriteString(":\n")
	buf.WriteString("\t\treturn true\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\treturn false\n")
	buf.WriteString("}\n\n")
}

func renderStruct(
	buf *bytes.Buffer,
	cfg *config.ModelsConfig,
	table *catalog.Table,
	mapper typemap.TypeMapper,
	enums []*catalog.Enum,
	mapOpts typemap.Options,
	sqlcOverrides *config.SqlcOverridesConfig,
	sqlcDefaults *config.SqlcDefaultsConfig,
) {
	// Singularize table name: todos → Todo, notes → Note
	structName := toCamelCase(inflection.Singular(table.Name))

	if table.Comment != "" {
		buf.WriteString(fmt.Sprintf("// %s %s\n", structName, table.Comment))
	}

	buf.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, col := range table.Columns {
		fieldName := toCamelCase(col.Name)
		fieldType := resolveType(cfg, table.Name, col, mapper, enums, mapOpts, sqlcOverrides)
		tags := buildTags(cfg, table.Name, col, sqlcOverrides, sqlcDefaults)

		tagStr := ""
		if len(tags) > 0 {
			tagStr = fmt.Sprintf(" `%s`", strings.Join(tags, " "))
		}

		buf.WriteString(fmt.Sprintf("\t%s %s%s\n", fieldName, fieldType, tagStr))
	}

	buf.WriteString("}\n\n")
}

// resolveType checks sqlc overrides → models type_overrides → default typemap.
func resolveType(
	cfg *config.ModelsConfig,
	tableName string,
	col *catalog.Column,
	mapper typemap.TypeMapper,
	enums []*catalog.Enum,
	mapOpts typemap.Options,
	sqlcOverrides *config.SqlcOverridesConfig,
) string {
	// 1. Check sqlc column overrides (highest priority)
	if sqlcOverrides != nil {
		columnKey := tableName + "." + col.Name
		for _, co := range sqlcOverrides.Columns {
			if co.Column == columnKey && co.GoType != nil {
				return extractGoType(co.GoType)
			}
		}
		// Check sqlc type overrides
		// Match nullable override for nullable columns, non-nullable for NOT NULL columns
		isNullable := !col.NotNull
		for _, to := range sqlcOverrides.Types {
			if strings.EqualFold(to.DbType, col.Type) && to.Nullable == isNullable {
				return extractGoType(to.GoType)
			}
		}
		// Fallback: if no exact nullable match, try non-nullable override for NOT NULL columns
		if !isNullable {
			for _, to := range sqlcOverrides.Types {
				if strings.EqualFold(to.DbType, col.Type) && !to.Nullable {
					return extractGoType(to.GoType)
				}
			}
		}
	}

	// 2. Check models.type_overrides from config
	for _, override := range cfg.TypeOverrides {
		if strings.EqualFold(override.SqlType, col.Type) {
			return override.GoType
		}
	}

	// 3. Default typemap
	goType := mapper.GoType(col, enums, mapOpts)

	// 4. When replace_nullable_types is set, convert pgtype.* to pointer types for nullable columns
	if !col.NotNull && mapOpts.EmitPointersForNull {
		goType = replaceNullablePgtype(goType)
	}

	return goType
}

// pgtypeToPointer maps pgtype types to their pointer equivalents.
var pgtypeToPointer = map[string]string{
	"pgtype.Text":        "*string",
	"pgtype.Int2":        "*int16",
	"pgtype.Int4":        "*int32",
	"pgtype.Int8":        "*int64",
	"pgtype.Float4":      "*float32",
	"pgtype.Float8":      "*float64",
	"pgtype.Bool":        "*bool",
	"pgtype.Timestamp":   "*time.Time",
	"pgtype.Timestamptz": "*time.Time",
	"pgtype.Date":        "*time.Time",
	"pgtype.UUID":        "*uuid.UUID",
	"pgtype.Numeric":     "*string",
}

func replaceNullablePgtype(goType string) string {
	if replacement, ok := pgtypeToPointer[goType]; ok {
		return replacement
	}
	return goType
}

// extractGoType converts sqlc go_type value (string or map) to a Go type string.
// Handles:
//   - "github.com/google/uuid.UUID" → "uuid.UUID"
//   - {"type": "JSONField", "import": "github.com/example/storecmn"} → "storecmn.JSONField"
//   - "string" → "string"
//   - {"type": "map[string]any"} → "map[string]any"
func extractGoType(v interface{}) string {
	switch t := v.(type) {
	case string:
		// Handle full import path like "github.com/google/uuid.UUID"
		if idx := strings.LastIndex(t, "."); idx > 0 {
			prefix := t[:idx]
			typeName := t[idx+1:]
			// Check if prefix looks like an import path (contains /)
			if strings.Contains(prefix, "/") {
				parts := strings.Split(prefix, "/")
				pkg := parts[len(parts)-1]
				return pkg + "." + typeName
			}
		}
		return t
	case map[string]interface{}:
		typeName, _ := t["type"].(string)
		importPath, _ := t["import"].(string)
		if importPath != "" && typeName != "" {
			parts := strings.Split(importPath, "/")
			pkg := parts[len(parts)-1]
			return pkg + "." + typeName
		}
		return typeName
	}
	return fmt.Sprintf("%v", v)
}

// buildTags builds struct tags for a column.
// emitDb/emitJson from sqlcDefaults are used as fallback if models config doesn't set them.
func buildTags(cfg *config.ModelsConfig, tableName string, col *catalog.Column, sqlcOverrides *config.SqlcOverridesConfig, sqlcDefaults *config.SqlcDefaultsConfig) []string {
	var tags []string

	emitDb := cfg.EmitDbTags
	emitJson := cfg.EmitJsonTags
	// Fallback to sqlc defaults if models config doesn't set them
	if sqlcDefaults != nil {
		if !emitDb && sqlcDefaults.EmitDbTags {
			emitDb = true
		}
		if !emitJson && sqlcDefaults.EmitJsonTags {
			emitJson = true
		}
	}

	if emitDb {
		tags = append(tags, fmt.Sprintf(`db:"%s"`, col.Name))
	}
	if emitJson {
		tags = append(tags, fmt.Sprintf(`json:"%s"`, col.Name))
	}

	// sqlc column overrides: go_struct_tag
	if sqlcOverrides != nil {
		columnKey := tableName + "." + col.Name
		for _, co := range sqlcOverrides.Columns {
			if co.Column == columnKey && co.GoStructTag != "" {
				tags = append(tags, co.GoStructTag)
			}
		}
	}

	// Custom tags from config
	for _, ct := range cfg.CustomTags {
		if col.NotNull && ct.Format != "" {
			tags = append(tags, fmt.Sprintf(`%s:"%s"`, ct.Name, ct.Format))
		}
	}

	return tags
}

// toCamelCase converts snake_case to CamelCase with Go acronym handling.
// Examples: user_id → UserID, http_url → HTTPURL, name → Name
func toCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	var result strings.Builder
	for _, part := range parts {
		lower := strings.ToLower(part)
		if acronym, ok := acronyms[lower]; ok {
			result.WriteString(acronym)
		} else {
			result.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return result.String()
}
