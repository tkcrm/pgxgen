package sqlc

var types map[string]string = map[string]string{
	"sql.NullInt32":   "*int32",
	"sql.NullInt64":   "*int64",
	"sql.NullInt16":   "*int16",
	"sql.NullFloat64": "*float64",
	"sql.NullFloat32": "*float32",
	"sql.NullString":  "*string",
	"sql.NullBool":    "*bool",
	"sql.NullTime":    "*time.Time",
	//"pgtype.Numeric":  "decimal.Decimal",
	//"json.RawMessage":       "map[string]interface{}",
	//"pqtype.NullRawMessage": "map[string]interface{}",
}
